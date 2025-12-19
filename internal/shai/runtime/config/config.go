package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	expectedType    = "shai-sandbox"
	expectedVersion = 1
)

// Config represents the parsed .shai/config.yaml configuration.
type Config struct {
	Type      string                  `yaml:"type"`
	Version   int                     `yaml:"version"`
	Image     string                  `yaml:"image"`
	User      string                  `yaml:"user"`
	Workspace string                  `yaml:"workspace"`
	Resources map[string]*ResourceSet `yaml:"resources"`
	Apply     []ApplyRule             `yaml:"apply"`

	sourcePath string
	sourceDir  string
	resolved   []pathResources
}

// ResourceSet groups runtime resources (env vars, mounts, calls).
type ResourceSet struct {
	Vars         []VarMapping    `yaml:"vars"`
	Mounts       []Mount         `yaml:"mounts"`
	Calls        []Call          `yaml:"calls"`
	HTTP         []string        `yaml:"http"`
	Ports        []Port          `yaml:"ports"`
	RootCommands []string        `yaml:"root-commands"`
	Options      ResourceOptions `yaml:"options"`
}

// ResourceOptions contains optional resource set configuration.
type ResourceOptions struct {
	Privileged bool `yaml:"privileged"`
}

// VarMapping defines a host->container variable mapping.
type VarMapping struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

// Mount describes a host mount.
type Mount struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Mode   string `yaml:"mode"`
}

// Call exposes a host command inside the container.
type Call struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
	AllowedArgs string `yaml:"allowed-args"`

	allowedRx *regexp.Regexp
}

// AllowedArgsRegexp returns the compiled regex for additional arguments (may be nil).
func (c Call) AllowedArgsRegexp() *regexp.Regexp {
	return c.allowedRx
}

// Port identifies an allow-listed network endpoint.
type Port struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// ApplyRule maps a workspace path to resource set names.
type ApplyRule struct {
	Path      string   `yaml:"path"`
	Resources []string `yaml:"resources"`
	Image     string   `yaml:"image"`
}

type pathResources struct {
	Path      string
	Resources []*ResolvedResource
	Image     string
}

// ResolvedResource couples a resource set with its name.
type ResolvedResource struct {
	Name string
	Spec *ResourceSet
}

// Load parses and validates a .shai/config.yaml file with template expansion.
func Load(path string, env map[string]string, vars map[string]string) (*Config, error) {
	if env == nil {
		env = map[string]string{}
	}
	if vars == nil {
		vars = map[string]string{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read shai config: %w", err)
	}
	return loadFromData(data, path, env, vars)
}

func (c *Config) applyTemplates(env, vars, conf map[string]string) error {
	var err error
	c.Image, err = expandTemplates(c.Image, env, vars, conf)
	if err != nil {
		return fmt.Errorf("image: %w", err)
	}
	c.User, err = expandTemplates(c.User, env, vars, conf)
	if err != nil {
		return fmt.Errorf("user: %w", err)
	}
	c.Workspace, err = expandTemplates(c.Workspace, env, vars, conf)
	if err != nil {
		return fmt.Errorf("workspace: %w", err)
	}

	for name, res := range c.Resources {
		for i := range res.Vars {
			res.Vars[i].Source, err = expandTemplates(res.Vars[i].Source, env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s var[%d] source: %w", name, i, err)
			}
			res.Vars[i].Target, err = expandTemplates(res.Vars[i].Target, env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s var[%d] target: %w", name, i, err)
			}
		}
		for i := range res.Mounts {
			res.Mounts[i].Source, err = expandTemplates(res.Mounts[i].Source, env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s mount[%d] source: %w", name, i, err)
			}
			res.Mounts[i].Target, err = expandTemplates(res.Mounts[i].Target, env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s mount[%d] target: %w", name, i, err)
			}
		}
		for i := range res.Calls {
			res.Calls[i].Description, err = expandTemplates(res.Calls[i].Description, env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s call[%d] description: %w", name, i, err)
			}
			res.Calls[i].Command, err = expandTemplates(res.Calls[i].Command, env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s call[%d] command: %w", name, i, err)
			}
		}
		for i := range res.HTTP {
			res.HTTP[i], err = expandTemplates(res.HTTP[i], env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s http[%d]: %w", name, i, err)
			}
		}
		for i := range res.Ports {
			res.Ports[i].Host, err = expandTemplates(res.Ports[i].Host, env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s port[%d] host: %w", name, i, err)
			}
		}
		for i := range res.RootCommands {
			res.RootCommands[i], err = expandTemplates(res.RootCommands[i], env, vars, conf)
			if err != nil {
				return fmt.Errorf("resource %s root-commands[%d]: %w", name, i, err)
			}
		}
	}
	for i := range c.Apply {
		c.Apply[i].Path, err = expandTemplates(c.Apply[i].Path, env, vars, conf)
		if err != nil {
			return fmt.Errorf("apply[%d] path: %w", i, err)
		}
		c.Apply[i].Image, err = expandTemplates(c.Apply[i].Image, env, vars, conf)
		if err != nil {
			return fmt.Errorf("apply[%d] image: %w", i, err)
		}
	}
	return nil
}

func (c *Config) validate() error {
	if c.Type != expectedType {
		return fmt.Errorf("unsupported config type %q (expected %q)", c.Type, expectedType)
	}
	if c.Version != expectedVersion {
		return fmt.Errorf("unsupported config version %d (expected %d)", c.Version, expectedVersion)
	}
	if strings.TrimSpace(c.Image) == "" {
		return errors.New("image is required")
	}
	// User and workspace now have defaults, so they're not required in config
	if len(c.Resources) == 0 {
		return errors.New("resources section is required")
	}
	for name, res := range c.Resources {
		for i := range res.Mounts {
			mode := strings.ToLower(strings.TrimSpace(res.Mounts[i].Mode))
			if mode == "" {
				mode = "ro"
			}
			if mode != "ro" && mode != "rw" {
				return fmt.Errorf("resource %s mount[%d] has invalid mode %q", name, i, res.Mounts[i].Mode)
			}
			res.Mounts[i].Mode = mode
		}
		for i := range res.Calls {
			if strings.TrimSpace(res.Calls[i].Name) == "" {
				return fmt.Errorf("resource %s call[%d] missing name", name, i)
			}
			if strings.TrimSpace(res.Calls[i].Command) == "" {
				return fmt.Errorf("resource %s call[%d] missing command", name, i)
			}
			if res.Calls[i].AllowedArgs != "" {
				rx, err := regexp.Compile(res.Calls[i].AllowedArgs)
				if err != nil {
					return fmt.Errorf("resource %s call[%s] invalid allowed-args regex: %w", name, res.Calls[i].Name, err)
				}
				res.Calls[i].allowedRx = rx
			}
		}
	}
	if len(c.Apply) == 0 {
		return errors.New("apply rules are required")
	}
	return nil
}

func (c *Config) resolvePaths() error {
	var resolved []pathResources
	for _, rule := range c.Apply {
		if len(rule.Resources) == 0 {
			return fmt.Errorf("apply path %q has no resources", rule.Path)
		}
		path := normalizePath(rule.Path)
		if path == "" {
			path = "."
		}
		image := strings.TrimSpace(rule.Image)
		if path == "." && image != "" {
			return fmt.Errorf("apply path %q cannot override image", rule.Path)
		}
		var resList []*ResolvedResource
		for _, name := range rule.Resources {
			res, ok := c.Resources[name]
			if !ok {
				return fmt.Errorf("apply path %q references unknown resource %q", rule.Path, name)
			}
			resList = append(resList, &ResolvedResource{Name: name, Spec: res})
		}
		resolved = append(resolved, pathResources{Path: path, Resources: resList, Image: image})
	}

	// Validate call uniqueness per path.
	for _, pr := range resolved {
		seen := map[string]string{}
		for _, res := range pr.Resources {
			for _, call := range res.Spec.Calls {
				if other, exists := seen[call.Name]; exists {
					return fmt.Errorf("call %q defined in both resources %s and %s for path %s", call.Name, other, res.Name, pr.Path)
				}
				seen[call.Name] = res.Name
			}
		}
	}

	c.resolved = resolved
	return nil
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "." {
		return "."
	}
	p = filepath.ToSlash(filepath.Clean(p))
	if strings.HasPrefix(p, "./") {
		p = strings.TrimPrefix(p, "./")
	}
	if p == "" {
		return "."
	}
	return p
}

func pathMatches(rulePath, candidate string) bool {
	if rulePath == "." {
		return true
	}
	if candidate == rulePath {
		return true
	}
	ruleParts := strings.Split(rulePath, "/")
	candidateParts := strings.Split(candidate, "/")
	if len(candidateParts) < len(ruleParts) {
		return false
	}
	for i := range ruleParts {
		if candidateParts[i] != ruleParts[i] {
			return false
		}
	}
	return true
}

// ResourcesForPath returns the resource sets applied to a workspace-relative path.
func (c *Config) ResourcesForPath(path string) []*ResolvedResource {
	candidate := normalizePath(path)
	var out []*ResolvedResource
	seen := make(map[string]bool)
	for _, pr := range c.resolved {
		if pathMatches(pr.Path, candidate) {
			for _, res := range pr.Resources {
				if seen[res.Name] {
					continue
				}
				seen[res.Name] = true
				out = append(out, res)
			}
		}
	}
	return out
}

// ImageForPath returns the first image override that matches the provided path.
func (c *Config) ImageForPath(path string) (string, bool) {
	candidate := normalizePath(path)
	var (
		image    string
		matched  bool
		matchLen int
	)
	for _, pr := range c.resolved {
		if pr.Image == "" {
			continue
		}
		if pathMatches(pr.Path, candidate) {
			length := 0
			if pr.Path != "." {
				length = len(strings.Split(pr.Path, "/"))
			}
			if !matched || length > matchLen {
				image = pr.Image
				matched = true
				matchLen = length
			}
		}
	}
	return image, matched
}

func loadFromData(data []byte, path string, env, vars map[string]string) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse shai config: %w", err)
	}
	cfg.sourcePath = path
	cfg.sourceDir = filepath.Dir(path)

	// Apply defaults for optional fields
	if strings.TrimSpace(cfg.User) == "" {
		cfg.User = "shai"
	}
	if strings.TrimSpace(cfg.Workspace) == "" {
		cfg.Workspace = "/src"
	}

	// First expand user and workspace fields (they can use env and vars but not conf)
	var err error
	emptyConf := map[string]string{}
	cfg.User, err = expandTemplates(cfg.User, env, vars, emptyConf)
	if err != nil {
		return nil, fmt.Errorf("user: %w", err)
	}
	cfg.Workspace, err = expandTemplates(cfg.Workspace, env, vars, emptyConf)
	if err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}

	// Build conf map with finalized user and workspace
	conf := map[string]string{
		"TARGET_USER": cfg.User,
		"WORKSPACE":   cfg.Workspace,
	}

	// Now expand all templates including conf references
	if err := cfg.applyTemplates(env, vars, conf); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if err := cfg.resolvePaths(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ResolveResources returns unique resource sets for the provided workspace-relative paths.
func (c *Config) ResolveResources(paths []string) []*ResolvedResource {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	seen := make(map[string]bool)
	var out []*ResolvedResource
	for _, path := range paths {
		for _, res := range c.ResourcesForPath(path) {
			if seen[res.Name] {
				continue
			}
			seen[res.Name] = true
			out = append(out, res)
		}
	}
	return out
}
