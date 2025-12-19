package config

import (
	"fmt"
	"regexp"
	"strings"
)

var templateExpr = regexp.MustCompile(`\${{\s*([^}]+)\s*}}`)

func expandTemplates(input string, env, vars, conf map[string]string) (string, error) {
	if input == "" {
		return "", nil
	}
	var expandErr error
	out := templateExpr.ReplaceAllStringFunc(input, func(match string) string {
		if expandErr != nil {
			return match
		}
		groups := templateExpr.FindStringSubmatch(match)
		if len(groups) != 2 {
			expandErr = fmt.Errorf("malformed template %q", match)
			return match
		}
		key := strings.TrimSpace(groups[1])
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			expandErr = fmt.Errorf("template %q missing scope", match)
			return match
		}
		scope, name := parts[0], parts[1]
		switch scope {
		case "env":
			if val, ok := env[name]; ok {
				return val
			}
			expandErr = fmt.Errorf("env %q not found for template %q", name, match)
			return match
		case "vars":
			if val, ok := vars[name]; ok {
				return val
			}
			expandErr = fmt.Errorf("var %q not found for template %q", name, match)
			return match
		case "conf":
			if val, ok := conf[name]; ok {
				return val
			}
			expandErr = fmt.Errorf("conf %q not found for template %q", name, match)
			return match
		default:
			expandErr = fmt.Errorf("unknown template scope %q in %q", scope, match)
			return match
		}
	})
	if expandErr != nil {
		return "", expandErr
	}
	if unresolved := templateExpr.FindString(out); unresolved != "" {
		return "", fmt.Errorf("unresolved template %q", unresolved)
	}
	return out, nil
}
