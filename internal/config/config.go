package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	App   AppConfig
	Watch []WatchEntry
	Rules []RuleEntry
}

type AppConfig struct {
	DryRun        bool
	DatabasePath  string
	Notifications bool
	SettleDelayMS int
}

type WatchEntry struct {
	Path      string
	Recursive bool
}

type RuleEntry struct {
	Name        string
	Enabled     bool
	Extensions  []string
	Regex       string
	Destination string
	Rename      string
}

func Default() *Config {
	return &Config{
		App: AppConfig{
			DryRun:        true,
			DatabasePath:  "~/.local/share/foldryn/history.jsonl",
			Notifications: true,
			SettleDelayMS: 1500,
		},
	}
}

// Load parses the small TOML subset used by Foldryn configs.
// It intentionally avoids external dependencies so the Linux daemon can build
// with the Go standard library only.
func Load(path string) (*Config, error) {
	file, err := os.Open(Expand(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := Default()
	section := ""
	currentWatch := -1
	currentRule := -1

	s := bufio.NewScanner(file)
	for lineNo := 1; s.Scan(); lineNo++ {
		line := stripComment(strings.TrimSpace(s.Text()))
		if line == "" {
			continue
		}
		switch line {
		case "[app]":
			section = "app"
			currentWatch = -1
			currentRule = -1
			continue
		case "[[watch]]":
			section = "watch"
			cfg.Watch = append(cfg.Watch, WatchEntry{})
			currentWatch = len(cfg.Watch) - 1
			currentRule = -1
			continue
		case "[[rules]]":
			section = "rules"
			cfg.Rules = append(cfg.Rules, RuleEntry{Enabled: true, Rename: "{basename}{ext}"})
			currentRule = len(cfg.Rules) - 1
			currentWatch = -1
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("config line %d: expected key = value", lineNo)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch section {
		case "app":
			if err := setApp(&cfg.App, key, val); err != nil {
				return nil, fmt.Errorf("config line %d: %w", lineNo, err)
			}
		case "watch":
			if currentWatch < 0 {
				return nil, fmt.Errorf("config line %d: watch value before [[watch]]", lineNo)
			}
			if err := setWatch(&cfg.Watch[currentWatch], key, val); err != nil {
				return nil, fmt.Errorf("config line %d: %w", lineNo, err)
			}
		case "rules":
			if currentRule < 0 {
				return nil, fmt.Errorf("config line %d: rule value before [[rules]]", lineNo)
			}
			if err := setRule(&cfg.Rules[currentRule], key, val); err != nil {
				return nil, fmt.Errorf("config line %d: %w", lineNo, err)
			}
		default:
			return nil, fmt.Errorf("config line %d: value outside a section", lineNo)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func setApp(a *AppConfig, key, val string) error {
	switch key {
	case "dry_run":
		v, err := parseBool(val)
		a.DryRun = v
		return err
	case "database_path", "history_path":
		v, err := parseString(val)
		a.DatabasePath = v
		return err
	case "notifications":
		v, err := parseBool(val)
		a.Notifications = v
		return err
	case "settle_delay_ms":
		v, err := strconv.Atoi(strings.TrimSpace(val))
		a.SettleDelayMS = v
		return err
	default:
		return fmt.Errorf("unknown app key %q", key)
	}
}

func setWatch(w *WatchEntry, key, val string) error {
	switch key {
	case "path":
		v, err := parseString(val)
		w.Path = v
		return err
	case "recursive":
		v, err := parseBool(val)
		w.Recursive = v
		return err
	default:
		return fmt.Errorf("unknown watch key %q", key)
	}
}

func setRule(r *RuleEntry, key, val string) error {
	switch key {
	case "name":
		v, err := parseString(val)
		r.Name = v
		return err
	case "enabled":
		v, err := parseBool(val)
		r.Enabled = v
		return err
	case "extensions":
		v, err := parseStringArray(val)
		r.Extensions = v
		return err
	case "regex":
		v, err := parseString(val)
		r.Regex = v
		return err
	case "destination":
		v, err := parseString(val)
		r.Destination = v
		return err
	case "rename":
		v, err := parseString(val)
		r.Rename = v
		return err
	default:
		return fmt.Errorf("unknown rule key %q", key)
	}
}

func (c *Config) Validate() error {
	if len(c.Watch) == 0 {
		return errors.New("at least one [[watch]] entry is required")
	}
	if len(c.Rules) == 0 {
		return errors.New("at least one [[rules]] entry is required")
	}
	if c.App.DatabasePath == "" {
		c.App.DatabasePath = "~/.local/share/foldryn/history.jsonl"
	}
	if c.App.SettleDelayMS <= 0 {
		c.App.SettleDelayMS = 1500
	}
	for i, w := range c.Watch {
		if strings.TrimSpace(w.Path) == "" {
			return fmt.Errorf("watch[%d].path is required", i)
		}
	}
	for i := range c.Rules {
		r := &c.Rules[i]
		if strings.TrimSpace(r.Name) == "" {
			return fmt.Errorf("rules[%d].name is required", i)
		}
		if strings.TrimSpace(r.Destination) == "" {
			return fmt.Errorf("rules[%d].destination is required", i)
		}
		if r.Rename == "" {
			r.Rename = "{basename}{ext}"
		}
	}
	return nil
}

func parseBool(v string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected boolean, got %q", v)
	}
}

func parseString(v string) (string, error) {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
		unq, err := strconv.Unquote(v)
		if err == nil {
			return unq, nil
		}
		return v[1 : len(v)-1], nil
	}
	return "", fmt.Errorf("expected quoted string, got %q", v)
}

func parseStringArray(v string) ([]string, error) {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return nil, fmt.Errorf("expected string array, got %q", v)
	}
	inner := strings.TrimSpace(v[1 : len(v)-1])
	if inner == "" {
		return nil, nil
	}
	var out []string
	var cur strings.Builder
	inQuote := rune(0)
	escaped := false
	for _, r := range inner {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && inQuote != 0 {
			cur.WriteRune(r)
			escaped = true
			continue
		}
		if inQuote != 0 {
			cur.WriteRune(r)
			if r == inQuote {
				inQuote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			inQuote = r
			cur.WriteRune(r)
			continue
		}
		if r == ',' {
			item, err := parseString(strings.TrimSpace(cur.String()))
			if err != nil {
				return nil, err
			}
			out = append(out, item)
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	if strings.TrimSpace(cur.String()) != "" {
		item, err := parseString(strings.TrimSpace(cur.String()))
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func stripComment(line string) string {
	inQuote := rune(0)
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inQuote != 0 {
			escaped = true
			continue
		}
		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			inQuote = r
			continue
		}
		if r == '#' {
			return strings.TrimSpace(line[:i])
		}
	}
	return line
}

func Expand(path string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return os.ExpandEnv(path)
}
