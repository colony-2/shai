package shai

import (
	"fmt"
	"strings"

	"github.com/colony-2/shai/internal/shai/runtime/alias"
	configpkg "github.com/colony-2/shai/internal/shai/runtime/config"
)

func resolvedResources(cfg *configpkg.Config, rwPaths []string, extraSets []string, prependSet *configpkg.ResourceSet, appendSet *configpkg.ResourceSet) ([]*configpkg.ResolvedResource, []string, string, error) {
	if cfg == nil {
		return nil, nil, "", nil
	}
	if err := configpkg.NormalizeResourceSet(prependSet, "prepend"); err != nil {
		return nil, nil, "", err
	}
	if err := configpkg.NormalizeResourceSet(appendSet, "append"); err != nil {
		return nil, nil, "", err
	}
	orderedPaths := orderedResourcePaths(rwPaths)
	base := cfg.ResolveResources(orderedPaths)
	image := selectImageOverride(cfg, orderedPaths)

	combined := make([]*configpkg.ResolvedResource, 0, len(extraSets)+len(base)+2)
	names := make([]string, 0, len(extraSets)+len(base))
	seen := make(map[string]bool)
	var missing []string

	if prependSet != nil {
		combined = append(combined, &configpkg.ResolvedResource{Name: "", Spec: prependSet})
	}

	for _, setName := range extraSets {
		name := strings.TrimSpace(setName)
		if name == "" {
			continue
		}
		res, ok := cfg.Resources[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if seen[name] {
			continue
		}
		combined = append(combined, &configpkg.ResolvedResource{Name: name, Spec: res})
		names = append(names, name)
		seen[name] = true
	}
	if len(missing) > 0 {
		return nil, nil, "", fmt.Errorf("unknown resource set(s): %s", strings.Join(missing, ", "))
	}

	for _, res := range base {
		if res == nil || res.Spec == nil {
			continue
		}
		if seen[res.Name] {
			continue
		}
		combined = append(combined, res)
		names = append(names, res.Name)
		seen[res.Name] = true
	}

	if appendSet != nil {
		combined = append(combined, &configpkg.ResolvedResource{Name: "", Spec: appendSet})
	}
	return combined, names, image, nil
}

func orderedResourcePaths(rwPaths []string) []string {
	seen := map[string]bool{".": true}
	paths := []string{"."}
	for _, p := range rwPaths {
		if p == "" {
			p = "."
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		paths = append(paths, p)
	}
	return paths
}

func callEntriesFromResources(resources []*configpkg.ResolvedResource) ([]*alias.Entry, error) {
	var entries []*alias.Entry
	seen := make(map[string]bool)
	for _, res := range resources {
		if res == nil || res.Spec == nil {
			continue
		}
		for _, callDef := range res.Spec.Calls {
			if seen[callDef.Name] {
				continue
			}
			entry, err := alias.NewEntry(callDef.Name, callDef.Description, callDef.Command, callDef.AllowedArgs)
			if err != nil {
				return nil, fmt.Errorf("invalid call %q: %w", callDef.Name, err)
			}
			entries = append(entries, entry)
			seen[callDef.Name] = true
		}
	}
	return entries, nil
}

func selectImageOverride(cfg *configpkg.Config, orderedPaths []string) string {
	if cfg == nil {
		return ""
	}
	for _, path := range orderedPaths {
		if path == "." {
			continue
		}
		if img, ok := cfg.ImageForPath(path); ok {
			return img
		}
	}
	return ""
}
