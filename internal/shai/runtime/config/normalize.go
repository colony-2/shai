package config

import (
	"fmt"
	"regexp"
	"strings"
)

// NormalizeResourceSet validates and normalizes a resource set definition.
func NormalizeResourceSet(res *ResourceSet, label string) error {
	if res == nil {
		return nil
	}
	prefix := "resource"
	if strings.TrimSpace(label) != "" {
		prefix = fmt.Sprintf("resource %s", label)
	}
	for i := range res.Mounts {
		mode := strings.ToLower(strings.TrimSpace(res.Mounts[i].Mode))
		if mode == "" {
			mode = "ro"
		}
		if mode != "ro" && mode != "rw" {
			return fmt.Errorf("%s mount[%d] has invalid mode %q", prefix, i, res.Mounts[i].Mode)
		}
		res.Mounts[i].Mode = mode
	}
	for i := range res.Calls {
		if strings.TrimSpace(res.Calls[i].Name) == "" {
			return fmt.Errorf("%s call[%d] missing name", prefix, i)
		}
		if strings.TrimSpace(res.Calls[i].Command) == "" {
			return fmt.Errorf("%s call[%d] missing command", prefix, i)
		}
		if res.Calls[i].AllowedArgs != "" {
			rx, err := regexp.Compile(res.Calls[i].AllowedArgs)
			if err != nil {
				return fmt.Errorf("%s call[%s] invalid allowed-args regex: %w", prefix, res.Calls[i].Name, err)
			}
			res.Calls[i].allowedRx = rx
		}
	}
	return nil
}
