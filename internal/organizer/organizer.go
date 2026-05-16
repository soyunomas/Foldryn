package organizer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/soyunomas/foldryn/internal/config"
	"github.com/soyunomas/foldryn/internal/database"
	"github.com/soyunomas/foldryn/internal/notifier"
	"github.com/soyunomas/foldryn/internal/rules"
)

type Organizer struct {
	Rules    []rules.CompiledRule
	DryRun   bool
	DB       *database.DB
	Notifier notifier.Notifier
}

func (o *Organizer) Handle(path string) error {
	if IsEphemeralPath(path) {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		if IsIgnorableMissing(err) {
			return nil
		}
		return err
	}
	if st.IsDir() {
		return nil
	}
	if st.Size() == 0 {
		// Ignorar archivos vacíos (0 bytes) que los navegadores crean como marcadores de posición
		// al iniciar una descarga. Cuando la descarga real termine, el archivo tendrá peso.
		return nil
	}
	rule := rules.Match(path, o.Rules)
	if rule == nil {
		return nil
	}
	target := renderTarget(path, rule.Entry)
	status := "moved"
	action := "move"
	var opErr error
	if o.DryRun {
		status = "dry-run"
	} else {
		opErr = moveFile(path, target)
		if opErr != nil {
			status = "error"
		}
	}
	if o.DB != nil {
		errText := ""
		if opErr != nil {
			errText = opErr.Error()
		}
		_ = o.DB.Insert(database.Event{Action: action, RuleName: rule.Entry.Name, Source: path, Target: target, Status: status, Error: errText})
	}
	if opErr != nil {
		return opErr
	}
	o.Notifier.Send("Foldryn", fmt.Sprintf("%s: %s", status, filepath.Base(path)))
	return nil
}

func renderTarget(source string, rule config.RuleEntry) string {
	now := time.Now()
	base := filepath.Base(source)
	ext := filepath.Ext(base)
	basename := strings.TrimSuffix(base, ext)
	repl := map[string]string{
		"{year}":     now.Format("2006"),
		"{month}":    now.Format("01"),
		"{day}":      now.Format("02"),
		"{basename}": basename,
		"{ext}":      ext,
	}
	dir := config.Expand(rule.Destination)
	name := rule.Rename
	if name == "" {
		name = "{basename}{ext}"
	}
	for k, v := range repl {
		dir = strings.ReplaceAll(dir, k, v)
		name = strings.ReplaceAll(name, k, v)
	}
	return filepath.Join(dir, sanitizeFileName(name))
}

func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\x00", "")
	return name
}

func moveFile(src, dst string) error {
	if filepath.Clean(src) == filepath.Clean(dst) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	final := uniquePath(dst)
	if err := os.Rename(src, final); err == nil {
		return nil
	}
	return copyThenDelete(src, final)
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i < 10000; i++ {
		cand := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(cand); errors.Is(err, os.ErrNotExist) {
			return cand
		}
	}
	return fmt.Sprintf("%s-%d%s", base, time.Now().UnixNano(), ext)
}

func copyThenDelete(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}
	return os.Remove(src)
}

func IsIgnorableMissing(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) || strings.Contains(strings.ToLower(err.Error()), "no such file or directory")
}

func IsEphemeralPath(path string) bool {
	name := filepath.Base(path)
	lower := strings.ToLower(name)

	if name == "" || name == "." || name == ".." {
		return true
	}
	if strings.HasPrefix(name, ".goutputstream-") || strings.HasPrefix(name, ".~") {
		return true
	}
	if strings.HasPrefix(name, ".") && (strings.HasSuffix(name, ".swp") || strings.HasSuffix(name, ".swx")) {
		return true
	}

	suffixes := []string{
		".part",       // Firefox, aria2, wget-style temporary downloads
		".crdownload", // Chromium temporary downloads
		".tmp",
		".temp",
		".download",
		".opdownload",
		".partial",
		".filepart",
		".kate-swp",
		"~",
	}
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}
