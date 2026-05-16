package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type DB struct{ path string }

type Event struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Action    string    `json:"action"`
	RuleName  string    `json:"rule_name"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	return &DB{path: path}, nil
}

func (db *DB) Close() error { return nil }

func (db *DB) Insert(e Event) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	e.ID = time.Now().UnixNano()
	f, err := os.OpenFile(db.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

func (db *DB) Recent(limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 20
	}
	f, err := os.Open(db.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []Event
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		all = append(all, e)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	start := len(all) - limit
	if start < 0 {
		start = 0
	}
	out := make([]Event, 0, len(all)-start)
	for i := len(all) - 1; i >= start; i-- {
		out = append(out, all[i])
	}
	return out, nil
}

func (e Event) String() string {
	return fmt.Sprintf("%s %s %s -> %s [%s]", e.CreatedAt.Format(time.RFC3339), e.RuleName, e.Source, e.Target, e.Status)
}
