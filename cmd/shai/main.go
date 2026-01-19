package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/colony-2/shai/internal/shai/runtime/config"
	"github.com/colony-2/shai/pkg/shai"
	"github.com/spf13/cobra"
)

const workspacePath = "/src"

// Version information set by GoReleaser at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Args = normalizeLegacyArgs(os.Args)
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		readWritePaths []string
		configPath     string
		templatePairs  []string
		resourceSets   []string
		imageOverride  string
		userOverride   string
		containerName  string
		privileged     bool
		verbose        bool
		noTTY          bool
	)

	cmd := &cobra.Command{
		Use:           "shai [--read-write <path>] [flags] [-- command ...]",
		Short:         "Launch an ephemeral shai sandbox with optional writable mounts",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			varMap, err := parseTemplateVars(templatePairs)
			if err != nil {
				return err
			}

			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			var postExec *shai.SandboxExec
			if len(args) > 0 {
				postExec = &shai.SandboxExec{
					Command: args,
					Workdir: workspacePath,
					UseTTY:  !noTTY,
				}
			}

			ctx, cancel := setupSignals()
			defer cancel()

			if err := runEphemeral(ctx, workingDir, readWritePaths, verbose, postExec, configPath, varMap, resourceSets, imageOverride, userOverride, privileged); err != nil {
				return err
			}

			_ = containerName // Flag retained for future naming support.
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringArrayVar(&readWritePaths, "read-write", nil, "Path to mount read-write (repeatable, alias: -rw)")
	flags.StringVarP(&configPath, "config", "c", "", fmt.Sprintf("Path to Shai config (default: <workspace>/%s)", shai.DefaultConfigRelPath))
	flags.StringArrayVar(&resourceSets, "resource-set", nil, "Resource set to activate (repeatable, alias: -rs)")
	flags.StringArrayVarP(&templatePairs, "var", "v", nil, fmt.Sprintf("Template variable for %s (key=value)", shai.DefaultConfigRelPath))
	flags.StringVarP(&imageOverride, "image", "i", "", "Override container image (highest precedence)")
	flags.StringVarP(&userOverride, "user", "u", "", "Override target user (highest precedence)")
	flags.StringVarP(&containerName, "name", "n", "", "Container name (optional)")
	flags.BoolVar(&privileged, "privileged", false, "Run container in privileged mode")
	flags.BoolVarP(&verbose, "verbose", "V", false, "Enable verbose logging")
	flags.BoolVarP(&noTTY, "no-tty", "T", false, "Disable TTY for post-setup command")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newGenerateCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("shai %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}
}

func newGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate default config file",
		Long:  "Generate a default .shai/config.yaml file in the current directory. Errors if the file already exists.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateDefaultConfig()
		},
	}
}

func parseTemplateVars(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	vars := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid var %q (expected key=value)", pair)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid var %q (empty key)", pair)
		}
		vars[key] = parts[1]
	}
	return vars, nil
}

func normalizeLegacyArgs(args []string) []string {
	const (
		rwAlias = "-rw"
		rsAlias = "-rs"
	)

	out := make([]string, len(args))
	copy(out, args)

	for i, arg := range out {
		switch {
		case arg == rwAlias:
			out[i] = "--read-write"
		case strings.HasPrefix(arg, rwAlias+"="):
			out[i] = "--read-write" + arg[len(rwAlias):]
		case arg == rsAlias:
			out[i] = "--resource-set"
		case strings.HasPrefix(arg, rsAlias+"="):
			out[i] = "--resource-set" + arg[len(rsAlias):]
		}
	}
	return out
}

func runEphemeral(ctx context.Context, workingDir string, rwPaths []string, verbose bool, postExec *shai.SandboxExec, configPath string, vars map[string]string, resourceSets []string, imageOverride, userOverride string, privileged bool) error {
	sandbox, err := shai.NewSandbox(shai.SandboxConfig{
		WorkingDir:       workingDir,
		ConfigFile:       configPath,
		TemplateVars:     vars,
		ReadWritePaths:   rwPaths,
		ResourceSets:     resourceSets,
		Verbose:          verbose,
		PostSetupExec:    postExec,
		ImageOverride:    imageOverride,
		UserOverride:     userOverride,
		Privileged:       privileged,
		ShowProgress:     true,
		ShowScriptOutput: true,
	})
	if err != nil {
		return err
	}
	defer sandbox.Close()

	return sandbox.Run(ctx)
}

func generateDefaultConfig() error {
	configDir := shai.ConfigDirName
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check config file: %w", err)
	}

	// Create .shai directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", configDir, err)
	}

	// Get default config content
	defaultConfig := config.GetDefaultConfigBytes()

	// Write config file
	if err := os.WriteFile(configPath, defaultConfig, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Generated default config at %s\n", configPath)
	return nil
}

// setupSignals configures signal handling and returns a cancellable context.
// In ephemeral mode, SIGINT is ignored so Ctrl-C reaches the container shell.
func setupSignals() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGINT)
	signal.Notify(sigCh, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()
	return ctx, cancel
}
