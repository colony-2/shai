package shai

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/colony-2/shai/internal/shai/runtime/alias"
	"github.com/colony-2/shai/internal/shai/runtime/bootstrap"
	configpkg "github.com/colony-2/shai/internal/shai/runtime/config"
	"github.com/docker/docker/api/types/container"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moby/term"
)

// EphemeralConfig represents configuration for ephemeral container execution.
type EphemeralConfig struct {
	WorkingDir          string
	ConfigFile          string
	TemplateVars        map[string]string
	ReadWritePaths      []string
	ResourceSets        []string
	PrependResourceSet  *configpkg.ResourceSet
	AppendResourceSet   *configpkg.ResourceSet
	Verbose             bool
	PostSetupExec       *ExecSpec
	Stdout              io.Writer
	Stderr              io.Writer
	GracefulStopTimeout time.Duration
	ImageOverride       string
	UserOverride        string
	HostUID             string
	HostGID             string
	Privileged          bool
	ShowProgress        bool
	ShowScriptOutput    bool
}

// ExecSpec describes a command to run post-setup.
type ExecSpec struct {
	Command []string
	Env     map[string]string
	Workdir string
	UseTTY  bool
}

// EphemeralRunner launches ephemeral containers using .shai/config.yaml.
type EphemeralRunner struct {
	config             EphemeralConfig
	shaiConfig         *configpkg.Config
	resources          []*configpkg.ResolvedResource
	resourceNames      []string
	image              string
	workspace          string
	docker             *client.Client
	mountBuilder       *MountBuilder
	aliasSvc           *alias.Service
	currentContainerID string
	hostEnv            map[string]string
	hostUID            string
	hostGID            string
	bootstrapDir       string
	bootstrapMount     string
	dockerHostAddr     string
}

func (r *EphemeralRunner) workspaceDir() string {
	if ws := strings.TrimSpace(r.workspace); ws != "" {
		return ws
	}
	if r.shaiConfig != nil {
		return strings.TrimSpace(r.shaiConfig.Workspace)
	}
	return ""
}

// NewEphemeralRunner creates a new ephemeral runner.
func NewEphemeralRunner(cfg EphemeralConfig) (*EphemeralRunner, error) {
	if cfg.WorkingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		cfg.WorkingDir = wd
	}

	if !cfg.Verbose && os.Getenv("SHAI_FORCE_VERBOSE") == "1" {
		cfg.Verbose = true
	}

	hostEnv := hostEnvMap()
	if strings.TrimSpace(cfg.HostUID) == "" || strings.TrimSpace(cfg.HostGID) == "" {
		uid, gid := hostUserIDs()
		if strings.TrimSpace(cfg.HostUID) == "" {
			cfg.HostUID = uid
		}
		if strings.TrimSpace(cfg.HostGID) == "" {
			cfg.HostGID = gid
		}
	}
	configPath := cfg.ConfigFile
	if configPath == "" {
		configPath = filepath.Join(cfg.WorkingDir, DefaultConfigRelPath)
	}
	shaiCfg, _, err := configpkg.LoadOrDefault(configPath, hostEnv, cfg.TemplateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to load shai config: %w", err)
	}

	dockerClient, err := newDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	mountBuilder, err := NewMountBuilder(cfg.WorkingDir, cfg.ReadWritePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to create mount builder: %w", err)
	}

	workspace := effectiveWorkspace(shaiCfg.Workspace, mountBuilder.ReadWritePaths)
	shaiCfg.Workspace = workspace

	resources, resourceNames, applyImageOverride, err := resolvedResources(shaiCfg, mountBuilder.ReadWritePaths, cfg.ResourceSets, cfg.PrependResourceSet, cfg.AppendResourceSet)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve resources: %w", err)
	}
	callEntries, err := callEntriesFromResources(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve calls: %w", err)
	}

	mcpBindAddr := getMCPServerBindAddr(context.Background(), dockerClient)
	dockerHostAddr := getDockerHostAddress()
	aliasSvc, err := alias.MaybeStart(alias.Config{
		WorkingDir:     cfg.WorkingDir,
		ShellPath:      os.Getenv("SHELL"),
		Debug:          os.Getenv("SHAI_ALIAS_DEBUG") != "",
		Entries:        callEntries,
		DockerHostAddr: dockerHostAddr,
		MCPBindAddr:    mcpBindAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize alias service: %w", err)
	}

	image, imageSource := chooseImage(shaiCfg.Image, cfg.ImageOverride, applyImageOverride)
	if cfg.Verbose {
		switch imageSource {
		case "cli":
			fmt.Fprintf(os.Stderr, "shai: using image override from flag: %s\n", image)
		case "apply":
			fmt.Fprintf(os.Stderr, "shai: using image override from apply rules: %s\n", image)
		}
	}

	runner := &EphemeralRunner{
		config:         cfg,
		shaiConfig:     shaiCfg,
		resources:      resources,
		resourceNames:  resourceNames,
		image:          image,
		workspace:      workspace,
		docker:         dockerClient,
		mountBuilder:   mountBuilder,
		aliasSvc:       aliasSvc,
		hostEnv:        hostEnv,
		hostUID:        cfg.HostUID,
		hostGID:        cfg.HostGID,
		dockerHostAddr: dockerHostAddr,
	}
	if cfg.Verbose {
		if len(resourceNames) > 0 {
			fmt.Fprintf(os.Stderr, "shai: activating resource sets: %s\n", strings.Join(resourceNames, ", "))
		} else {
			fmt.Fprintln(os.Stderr, "shai: no resource sets activated")
		}
	}
	return runner, nil
}

// Run creates and runs the container to completion.
func (r *EphemeralRunner) Run(ctx context.Context) error {
	useTTY := r.shouldUseTTY()
	return r.runEphemeralContainer(ctx, useTTY)
}

// Start launches the container and returns a session for supervision.
func (r *EphemeralRunner) Start(ctx context.Context) (*Session, error) {
	useTTY := r.shouldUseTTY()

	sctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	idCh := make(chan string, 1)
	go func() {
		done <- r.runEphemeralContainerWithID(sctx, useTTY, idCh)
	}()

	var cid string
	select {
	case cid = <-idCh:
		r.currentContainerID = cid
	case err := <-done:
		cancel()
		return nil, err
	case <-time.After(10 * time.Second):
		cancel()
		return nil, errors.New("timeout creating container")
	}

	return &Session{
		ContainerID: cid,
		waitCh:      done,
		cancel:      cancel,
		docker:      r.docker,
		timeout:     r.config.GracefulStopTimeout,
	}, nil
}

// Close cleans up resources.
func (r *EphemeralRunner) Close() error {
	if r.aliasSvc != nil {
		r.aliasSvc.Close()
	}
	if r.bootstrapDir != "" {
		_ = os.RemoveAll(r.bootstrapDir)
		r.bootstrapDir = ""
		r.bootstrapMount = ""
	}
	if r.docker != nil {
		return r.docker.Close()
	}
	return nil
}

// GetContainerID returns the current container ID (primarily for tests).
func (r *EphemeralRunner) GetContainerID() string {
	return r.currentContainerID
}

func (r *EphemeralRunner) shouldUseTTY() bool {
	if r.config.PostSetupExec != nil {
		return r.config.PostSetupExec.UseTTY
	}
	return true
}

func (r *EphemeralRunner) runEphemeralContainer(ctx context.Context, useTTY bool) error {
	idCh := make(chan string, 1)
	return r.runEphemeralContainerWithID(ctx, useTTY, idCh)
}

func (r *EphemeralRunner) runEphemeralContainerWithID(ctx context.Context, useTTY bool, idCh chan<- string) error {
	containerName := generateContainerName()

	containerCfg, hostCfg, err := r.buildDockerConfigs(useTTY, containerName)
	if err != nil {
		return err
	}

	if err := r.ensureImage(ctx, containerCfg.Image); err != nil {
		return err
	}

	resp, err := r.docker.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	r.currentContainerID = resp.ID

	if err := r.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	select {
	case idCh <- resp.ID:
	default:
	}

	attachOpts := container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}
	hijacked, err := r.docker.ContainerAttach(ctx, resp.ID, attachOpts)
	if err != nil {
		return fmt.Errorf("attach container: %w", err)
	}
	defer hijacked.Close()

	stdinFD := os.Stdin.Fd()
	interactiveTTY := useTTY && term.IsTerminal(stdinFD)

	var resizeStop func()
	if interactiveTTY {
		if st, err := term.MakeRaw(stdinFD); err == nil {
			defer term.RestoreTerminal(stdinFD, st)
		}
		resizeStop = r.startTTYResizeWatcher(ctx, stdinFD, resp.ID)
	}
	if resizeStop != nil {
		defer resizeStop()
	}

	var ctrlFilter *ctrlCFilter
	stdinReader := io.Reader(os.Stdin)
	if interactiveTTY {
		ctrlFilter = newCtrlCFilter(os.Stdin)
		stdinReader = ctrlFilter
	}

	enableCtrlC := func() {}
	if ctrlFilter != nil {
		enableCtrlC = ctrlFilter.Enable
		defer ctrlFilter.Enable()
	}

	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(hijacked.Conn, stdinReader)
		errCh <- err
	}()

	startMarker := ""
	if r.config.ShowScriptOutput {
		startMarker = r.buildStartMarker()
	} else if ctrlFilter != nil {
		ctrlFilter.Enable()
	}

	if interactiveTTY {
		writer := newExecStartDetector(os.Stdout, startMarker, enableCtrlC)
		go func() {
			_, err := io.Copy(writer, hijacked.Conn)
			if closeErr := writer.Close(); err == nil {
				err = closeErr
			}
			errCh <- err
		}()
	} else if useTTY {
		writer := newExecStartDetector(os.Stdout, startMarker, nil)
		go func() {
			_, err := io.Copy(writer, hijacked.Conn)
			if closeErr := writer.Close(); err == nil {
				err = closeErr
			}
			errCh <- err
		}()
	} else {
		go func() {
			stdout := r.config.Stdout
			if stdout == nil {
				stdout = os.Stdout
			}
			stderr := r.config.Stderr
			if stderr == nil {
				stderr = os.Stderr
			}
			writer := newExecStartDetector(stdout, startMarker, nil)
			_, err := stdcopy.StdCopy(writer, stderr, hijacked.Reader)
			if closeErr := writer.Close(); err == nil {
				err = closeErr
			}
			errCh <- err
		}()
	}

	waitCh, errChWait := r.docker.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
	default:
	}

	var status container.WaitResponse
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errChWait:
		if err != nil {
			return err
		}
		status = <-waitCh
	case status = <-waitCh:
	}

	if status.Error != nil {
		return errors.New(status.Error.Message)
	}
	if status.StatusCode != 0 {
		return fmt.Errorf("container exited with status %d", status.StatusCode)
	}
	return nil
}

const (
	bootstrapConfigVersion = 1
)

// buildStartMarker constructs the exact bootstrap completion marker that the
// container will output. This marker is used to detect when the bootstrap
// script has completed and it's safe to enable ctrl-c passthrough.
func (r *EphemeralRunner) buildStartMarker() string {
	targetUser := r.shaiConfig.User
	if r.config.UserOverride != "" {
		targetUser = r.config.UserOverride
	}

	resourceSummary := ""
	if len(r.resourceNames) > 0 {
		resourceSummary = strings.Join(r.resourceNames, ", ")
	}

	return fmt.Sprintf("Shai sandbox started using [%s] as user [%s]. Resource sets: [%s]",
		r.image, targetUser, resourceSummary)
}

func (r *EphemeralRunner) buildDockerConfigs(useTTY bool, containerName string) (*container.Config, *container.HostConfig, error) {
	if err := r.ensureBootstrapScript(); err != nil {
		return nil, nil, err
	}

	bootstrapArgs, err := r.buildBootstrapArgs()
	if err != nil {
		return nil, nil, err
	}

	entrypoint := []string{"/shai-bootstrap/boot.sh"}

	env := []string{}
	if r.config.Verbose {
		env = append(env, "SHAI_VERBOSE=1")
	}
	if r.config.ShowScriptOutput {
		env = append(env, "SHAI_SCRIPT_OUTPUT=1")
	} else {
		env = append(env, "SHAI_SCRIPT_OUTPUT=0")
	}
	if r.aliasSvc != nil {
		env = append(env, r.aliasSvc.Env()...)
	}
	if strings.TrimSpace(r.hostUID) != "" {
		env = append(env, fmt.Sprintf("DEV_UID=%s", strings.TrimSpace(r.hostUID)))
	}
	if strings.TrimSpace(r.hostGID) != "" {
		env = append(env, fmt.Sprintf("DEV_GID=%s", strings.TrimSpace(r.hostGID)))
	}

	cfg := &container.Config{
		Image:        r.image,
		Hostname:     containerName,
		WorkingDir:   r.workspaceDir(),
		Entrypoint:   entrypoint,
		Cmd:          bootstrapArgs,
		User:         "root",
		Tty:          useTTY,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		Env:          env,
	}

	mounts := r.mountBuilder.BuildMounts()
	resourceMounts, err := r.resourceMounts()
	if err != nil {
		return nil, nil, err
	}
	mounts = append(mounts, resourceMounts...)
	mounts = append(mounts, mount.Mount{
		Type:     mount.TypeBind,
		Source:   r.bootstrapMount,
		Target:   "/shai-bootstrap",
		ReadOnly: false,
	})

	// Determine if container should run in privileged mode
	privileged := r.config.Privileged || r.hasPrivilegedResource()

	hostCfg := &container.HostConfig{
		AutoRemove: true,
		Mounts:     mounts,
		ExtraHosts: []string{fmt.Sprintf("%s:host-gateway", r.dockerHostAddr)},
		CapAdd:     []string{"NET_ADMIN"},
		Privileged: privileged,
	}
	return cfg, hostCfg, nil
}

// hasPrivilegedResource checks if any active resource set has options.privileged:true
func (r *EphemeralRunner) hasPrivilegedResource() bool {
	for _, res := range r.resources {
		if res.Spec != nil && res.Spec.Options.Privileged {
			return true
		}
	}
	return false
}

func (r *EphemeralRunner) buildBootstrapArgs() ([]string, error) {
	envMap, err := r.collectEnvMappings()
	if err != nil {
		return nil, err
	}

	exec := r.config.PostSetupExec
	httpList := uniqueHTTPHosts(r.resources)
	portList := uniquePortEntries(r.resources)
	rootCommands := collectRootCommands(r.resources)

	targetUser := r.shaiConfig.User
	if r.config.UserOverride != "" {
		targetUser = r.config.UserOverride
	}

	args := []string{
		"--version", strconv.Itoa(bootstrapConfigVersion),
		"--user", targetUser,
		"--workspace", r.workspaceDir(),
		"--rm", "true",
	}

	if img := strings.TrimSpace(r.image); img != "" {
		args = append(args, "--image-name", img)
	}
	for _, name := range r.resourceNames {
		if strings.TrimSpace(name) == "" {
			continue
		}
		args = append(args, "--resource-name", name)
	}

	for _, pair := range orderedKeyValuePairs(envMap) {
		args = append(args, "--exec-env", pair)
	}

	if exec != nil && len(exec.Command) > 0 {
		for _, arg := range exec.Command {
			args = append(args, "--exec-cmd", arg)
		}
	}
	if exec != nil && len(exec.Env) > 0 {
		for _, pair := range orderedKeyValuePairs(exec.Env) {
			args = append(args, "--exec-env", pair)
		}
	}

	for _, host := range httpList {
		args = append(args, "--http-allow", host)
	}
	for _, entry := range portList {
		args = append(args, "--port-allow", entry)
	}

	for _, cmd := range rootCommands {
		args = append(args, "--root-cmd", cmd)
	}

	if r.config.Verbose {
		args = append(args, "--verbose")
	}

	return args, nil
}

func (r *EphemeralRunner) collectEnvMappings() (map[string]string, error) {
	envs := map[string]string{}
	for _, res := range r.resources {
		if res == nil || res.Spec == nil {
			continue
		}
		for _, vm := range res.Spec.Vars {
			source := strings.TrimSpace(vm.Source)
			if source == "" {
				return nil, errors.New("vars entry missing source")
			}
			value, ok := r.hostEnv[source]
			if !ok {
				return nil, fmt.Errorf("host env %q not set", source)
			}
			target := strings.TrimSpace(vm.Target)
			if target == "" {
				target = source
			}
			envs[target] = value
		}
	}
	return envs, nil
}

func (r *EphemeralRunner) resourceMounts() ([]mount.Mount, error) {
	var mounts []mount.Mount
	var skippedMounts []string
	for _, res := range r.resources {
		if res == nil || res.Spec == nil {
			continue
		}
		for _, m := range res.Spec.Mounts {
			source := m.Source
			if !filepath.IsAbs(source) {
				source = filepath.Join(r.config.WorkingDir, source)
			}
			if _, err := os.Stat(source); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					skippedMounts = append(skippedMounts, fmt.Sprintf("%s -> %s", source, m.Target))
					continue
				}
				return nil, fmt.Errorf("resource mount %s: %w", source, err)
			}
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   source,
				Target:   m.Target,
				ReadOnly: m.Mode != "rw",
			})
		}
	}

	// Print warning about skipped mounts
	if len(skippedMounts) > 0 && r.config.ShowScriptOutput {
		fmt.Fprintf(os.Stderr, "Warning: Skipped %d missing resource mount(s):\n", len(skippedMounts))
		for _, sm := range skippedMounts {
			fmt.Fprintf(os.Stderr, "  - %s\n", sm)
		}
	}

	return mounts, nil
}

func (r *EphemeralRunner) ensureImage(ctx context.Context, img string) error {
	if _, _, err := r.docker.ImageInspectWithRaw(ctx, img); err == nil {
		return nil
	}
	reader, err := r.docker.ImagePull(ctx, img, imagetypes.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", img, err)
	}
	defer reader.Close()

	if r.config.ShowProgress {
		return jsonmessage.DisplayJSONMessagesStream(
			reader,
			os.Stdout,
			os.Stdout.Fd(),
			term.IsTerminal(os.Stdout.Fd()),
			nil,
		)
	}

	_, _ = io.Copy(io.Discard, reader)
	return nil
}

// Session represents a started container.
type Session struct {
	ContainerID string
	waitCh      <-chan error
	cancel      context.CancelFunc
	docker      *client.Client
	timeout     time.Duration
}

// Wait blocks until the container exits.
func (s *Session) Wait(ctx context.Context) error {
	select {
	case err := <-s.waitCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop attempts to stop the running container gracefully.
func (s *Session) Stop(ctx context.Context) error {
	if s.ContainerID == "" {
		return nil
	}
	timeout := s.timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	stopCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return s.docker.ContainerStop(stopCtx, s.ContainerID, container.StopOptions{})
}

// Close cancels the supervising context.
func (s *Session) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

type ctrlCFilter struct {
	reader  io.Reader
	enabled atomic.Bool
}

func newCtrlCFilter(r io.Reader) *ctrlCFilter {
	return &ctrlCFilter{reader: r}
}

func (f *ctrlCFilter) Enable() {
	f.enabled.Store(true)
}

func (f *ctrlCFilter) Read(p []byte) (int, error) {
	for {
		n, err := f.reader.Read(p)
		if n == 0 {
			return n, err
		}
		if f.enabled.Load() {
			return n, err
		}
		out := p[:0]
		for _, b := range p[:n] {
			if b == 0x03 {
				continue
			}
			out = append(out, b)
		}
		if len(out) == 0 && err == nil {
			continue
		}
		copy(p, out)
		return len(out), err
	}
}

type execStartDetector struct {
	dst       io.Writer
	marker    []byte
	buf       []byte
	triggered bool
	onExec    func()
}

func newExecStartDetector(dst io.Writer, marker string, onExec func()) *execStartDetector {
	return &execStartDetector{
		dst:    dst,
		marker: []byte(marker),
		onExec: onExec,
	}
}

func (d *execStartDetector) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if _, err := d.dst.Write(p); err != nil {
		return 0, err
	}
	if d.triggered || len(d.marker) == 0 {
		return len(p), nil
	}

	d.buf = append(d.buf, p...)
	if len(d.buf) >= len(d.marker) {
		searchLimit := len(d.buf) - len(d.marker)
		for i := 0; i <= searchLimit; i++ {
			if bytes.Equal(d.buf[i:i+len(d.marker)], d.marker) {
				d.triggered = true
				if d.onExec != nil {
					d.onExec()
				}
				d.buf = nil
				return len(p), nil
			}
		}

		// Retain only the trailing substring needed for future matches.
		start := len(d.buf) - len(d.marker) + 1
		if start < 0 {
			start = 0
		}
		d.buf = append([]byte{}, d.buf[start:]...)
	}
	return len(p), nil
}

func (d *execStartDetector) Close() error {
	return nil
}

func (r *EphemeralRunner) startTTYResizeWatcher(ctx context.Context, fd uintptr, containerID string) func() {
	if !term.IsTerminal(fd) {
		return nil
	}
	resize := func() {
		if ws, err := term.GetWinsize(fd); err == nil && ws != nil {
			_ = r.docker.ContainerResize(context.Background(), containerID, container.ResizeOptions{
				Height: uint(ws.Height),
				Width:  uint(ws.Width),
			})
		}
	}
	resize()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)

	done := make(chan struct{})
	go func() {
		defer signal.Stop(sigCh)
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-sigCh:
				resize()
			}
		}
	}()

	return func() {
		close(done)
	}
}

func orderedKeyValuePairs(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, values[k]))
	}
	return pairs
}

func uniqueHTTPHosts(resources []*configpkg.ResolvedResource) []string {
	seen := make(map[string]bool)
	var hosts []string
	for _, res := range resources {
		if res == nil || res.Spec == nil {
			continue
		}
		for _, host := range res.Spec.HTTP {
			trimmed := strings.TrimSpace(host)
			if trimmed == "" || seen[trimmed] {
				continue
			}
			seen[trimmed] = true
			hosts = append(hosts, trimmed)
		}
	}
	sort.Strings(hosts)
	return hosts
}

func uniquePortEntries(resources []*configpkg.ResolvedResource) []string {
	seen := make(map[string]bool)
	var entries []string
	for _, res := range resources {
		if res == nil || res.Spec == nil {
			continue
		}
		for _, p := range res.Spec.Ports {
			host := strings.TrimSpace(p.Host)
			if host == "" || p.Port == 0 {
				continue
			}
			key := fmt.Sprintf("%s:%d", host, p.Port)
			if seen[key] {
				continue
			}
			seen[key] = true
			entries = append(entries, key)
		}
	}
	sort.Strings(entries)
	return entries
}

func collectRootCommands(resources []*configpkg.ResolvedResource) []string {
	var commands []string
	for _, res := range resources {
		if res == nil || res.Spec == nil {
			continue
		}
		for _, cmd := range res.Spec.RootCommands {
			trimmed := strings.TrimSpace(cmd)
			if trimmed == "" {
				continue
			}
			commands = append(commands, trimmed)
		}
	}
	return commands
}

func (r *EphemeralRunner) ensureBootstrapScript() error {
	if r.bootstrapMount != "" {
		return nil
	}
	id, err := randomHex(8)
	if err != nil {
		return fmt.Errorf("generate bootstrap id: %w", err)
	}
	baseDir := filepath.Join(os.TempDir(), "shai-"+id)
	scriptDir := filepath.Join(baseDir, "shai-bootstrap")
	if err := os.MkdirAll(scriptDir, 0o700); err != nil {
		return fmt.Errorf("create bootstrap dir: %w", err)
	}
	scriptPath := filepath.Join(scriptDir, "boot.sh")
	if err := os.WriteFile(scriptPath, bootstrap.Script, 0o700); err != nil {
		return fmt.Errorf("write bootstrap script: %w", err)
	}
	aliasPath := filepath.Join(scriptDir, "shai-remote")
	if err := os.WriteFile(aliasPath, bootstrap.AliasScript, 0o700); err != nil {
		return fmt.Errorf("write alias script: %w", err)
	}
	confDir := filepath.Join(scriptDir, "conf")
	if err := copyEmbeddedDir(bootstrap.ConfFS, "conf", confDir); err != nil {
		return fmt.Errorf("write bootstrap configs: %w", err)
	}
	r.bootstrapDir = baseDir
	r.bootstrapMount = scriptDir
	return nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func copyEmbeddedDir(fsys fs.FS, srcDir, destDir string) error {
	return fs.WalkDir(fsys, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			if rel == "." {
				return os.MkdirAll(destDir, 0o755)
			}
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func newDockerClient() (*client.Client, error) {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err == nil {
			return cli, nil
		}
		return nil, fmt.Errorf("DOCKER_HOST=%s: %w", host, err)
	}

	var errs []string
	for _, sock := range dockerSocketCandidates() {
		info, err := os.Stat(sock)
		if err != nil || info.Mode()&os.ModeSocket == 0 {
			continue
		}
		host := "unix://" + sock
		cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", host, err))
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, pingErr := cli.Ping(ctx)
		cancel()
		if pingErr != nil {
			errs = append(errs, fmt.Sprintf("%s ping: %v", host, pingErr))
			_ = cli.Close()
			continue
		}
		return cli, nil
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("unable to connect to docker: %s", strings.Join(errs, "; "))
	}
	return nil, errors.New("unable to find docker socket; set DOCKER_HOST or ensure Docker/Podman is running")
}

func dockerSocketCandidates() []string {
	seen := make(map[string]bool)
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
	}
	add("/var/run/docker.sock")
	add("/run/docker.sock")
	add("/var/run/podman/podman.sock")
	add("/run/podman/podman.sock")

	if home := os.Getenv("HOME"); home != "" {
		add(filepath.Join(home, ".docker", "run", "docker.sock"))
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		add(filepath.Join(xdg, "docker.sock"))
		add(filepath.Join(xdg, "podman", "podman.sock"))
	}
	if current, err := user.Current(); err == nil && current.Uid != "" {
		add(filepath.Join("/run/user", current.Uid, "docker.sock"))
		add(filepath.Join("/run/user", current.Uid, "podman/podman.sock"))
	} else if uid := os.Getenv("UID"); uid != "" {
		add(filepath.Join("/run/user", uid, "docker.sock"))
		add(filepath.Join("/run/user", uid, "podman/podman.sock"))
	}

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func effectiveWorkspace(base string, rwPaths []string) string {
	if len(rwPaths) != 1 {
		return base
	}
	rwPath := rwPaths[0]
	if rwPath == "" || rwPath == "." || filepath.IsAbs(rwPath) {
		return base
	}
	return path.Join(base, rwPath)
}

func chooseImage(defaultImage, cliOverride, applyOverride string) (string, string) {
	if img := strings.TrimSpace(cliOverride); img != "" {
		return img, "cli"
	}
	if img := strings.TrimSpace(applyOverride); img != "" {
		return img, "apply"
	}
	return defaultImage, ""
}

// getDockerHostAddress returns the hostname/address that containers should use to reach the host.
// On macOS/Windows, this is "host.docker.internal". On Linux, we use "host-gateway" which Docker
// will resolve to the gateway IP.
func getDockerHostAddress() string {
	// host.docker.internal works on macOS and Windows
	// On Linux with Docker 20.10+, we can use the special "host-gateway" value in ExtraHosts
	// which Docker automatically resolves to the gateway IP
	return "host.docker.internal"
}

// getDockerBridgeGatewayIP queries Docker to find the gateway IP of the default bridge network.
// This is used to determine what IP address the MCP server should bind to for container access.
func getDockerBridgeGatewayIP(ctx context.Context, dockerClient *client.Client) (string, error) {
	if dockerClient == nil {
		return "", fmt.Errorf("docker client is nil")
	}

	// Get the default bridge network
	networks, err := dockerClient.NetworkList(ctx, networktypes.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("list networks: %w", err)
	}

	// Look for the bridge network
	for _, network := range networks {
		if network.Name == "bridge" {
			if len(network.IPAM.Config) > 0 {
				gateway := network.IPAM.Config[0].Gateway
				if gateway != "" {
					return gateway, nil
				}
			}
		}
	}

	return "", fmt.Errorf("bridge network gateway not found")
}

// getMCPServerBindAddr determines what address the MCP server should bind to.
// On macOS/Windows (Docker Desktop), we use 127.0.0.1 since host.docker.internal
// works with localhost via Docker Desktop's VM networking.
// On Linux, we need to bind to the Docker bridge gateway IP so containers can reach it.
func getMCPServerBindAddr(ctx context.Context, dockerClient *client.Client) string {
	// On macOS and Windows, Docker Desktop handles host.docker.internal via VM networking
	// The MCP server should bind to localhost since the bridge gateway IP doesn't exist
	// on the host's network interfaces
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return "127.0.0.1:0"
	}

	// On Linux (native Docker), the bridge gateway IP exists on the host
	// Bind to it specifically for better security
	gatewayIP, err := getDockerBridgeGatewayIP(ctx, dockerClient)
	if err == nil && gatewayIP != "" {
		return gatewayIP + ":0"
	}

	// Fallback to binding on all interfaces if we can't determine the bridge IP
	// This is less secure but ensures functionality
	return "0.0.0.0:0"
}

// generateContainerName creates a container name with format "shai-<random>"
// where random is a short random hex string
func generateContainerName() string {
	randomBytes := make([]byte, 4) // 4 bytes = 8 hex chars
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-based name if random generation fails
		return fmt.Sprintf("shai-%d", time.Now().UnixNano())
	}
	return "shai-" + hex.EncodeToString(randomBytes)
}
