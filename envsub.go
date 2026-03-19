package loosey

import (
	"fmt"
	"strings"
)

func envToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		if idx := strings.IndexByte(e, '='); idx >= 0 {
			m[e[:idx]] = e[idx+1:]
		}
	}
	return m
}

func expandEnv(s string, env []string) (string, error) {
	envMap := envToMap(env)
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '$' {
			out.WriteByte(s[i])
			i++
			continue
		}
		// $ found
		if i+1 < len(s) && s[i+1] == '$' {
			// $$ escape
			out.WriteByte('$')
			i += 2
			continue
		}
		if i+1 < len(s) && s[i+1] == '{' {
			// ${...} expansion
			end := strings.IndexByte(s[i+2:], '}')
			if end < 0 {
				return "", fmt.Errorf("loosey: unclosed ${ in %q", s)
			}
			expr := s[i+2 : i+2+end]
			i = i + 2 + end + 1

			key := expr
			defaultVal := ""
			hasDefault := false
			if idx := strings.Index(expr, ":-"); idx >= 0 {
				key = expr[:idx]
				defaultVal = expr[idx+2:]
				hasDefault = true
			}

			val, ok := envMap[key]
			if !ok {
				if hasDefault {
					val = defaultVal
				} else {
					return "", fmt.Errorf("loosey: undefined environment variable %q", key)
				}
			}
			out.WriteString(val)
			continue
		}
		// bare $ not followed by { or $
		out.WriteByte(s[i])
		i++
	}
	return out.String(), nil
}
