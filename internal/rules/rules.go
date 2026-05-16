package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/soyunomas/foldryn/internal/config"
)

type CompiledRule struct {
	Entry config.RuleEntry
	Regex *regexp.Regexp
}

func Compile(entries []config.RuleEntry) ([]CompiledRule, error) {
	out := make([]CompiledRule, 0, len(entries))
	for _, e := range entries {
		if !e.Enabled {
			continue
		}
		cr := CompiledRule{Entry: e}
		if e.Regex != "" {
			rx, err := regexp.Compile(e.Regex)
			if err != nil {
				return nil, err
			}
			cr.Regex = rx
		}
		out = append(out, cr)
	}
	return out, nil
}

func Match(path string, compiled []CompiledRule) *CompiledRule {
	name := filepath.Base(path)
	ext := strings.ToLower(filepath.Ext(name))
	for i := range compiled {
		r := &compiled[i]
		if len(r.Entry.Extensions) > 0 {
			for _, allowed := range r.Entry.Extensions {
				if strings.EqualFold(ext, normalizeExt(allowed)) {
					return r
				}
			}
		}
		if r.Regex != nil && r.Regex.MatchString(name) {
			return r
		}
	}
	return nil
}

func normalizeExt(ext string) string {
	ext = strings.TrimSpace(strings.ToLower(ext))
	if ext == "" {
		return ext
	}
	if !strings.HasPrefix(ext, ".") {
		return "." + ext
	}
	return ext
}
