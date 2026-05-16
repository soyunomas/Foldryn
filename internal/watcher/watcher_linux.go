//go:build linux

package watcher

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/soyunomas/foldryn/internal/config"
	"github.com/soyunomas/foldryn/internal/organizer"
)

type Handler func(path string) error

type Watcher struct {
	entries []config.WatchEntry
	delay   time.Duration
	handle  Handler

	mu       sync.Mutex
	timers   map[string]*time.Timer
	wdToPath map[int]string
}

func New(entries []config.WatchEntry, delay time.Duration, handle Handler) *Watcher {
	if delay <= 0 {
		delay = 1500 * time.Millisecond
	}
	return &Watcher{entries: entries, delay: delay, handle: handle, timers: make(map[string]*time.Timer), wdToPath: make(map[int]string)}
}

func (w *Watcher) Run() error {
	fd, err := syscall.InotifyInit()
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	for _, e := range w.entries {
		root := config.Expand(e.Path)
		if e.Recursive {
			if err := w.addTree(fd, root); err != nil {
				return err
			}
		} else if err := w.addWatch(fd, root); err != nil {
			return err
		}
		log.Printf("watching %s", root)
	}

	buf := make([]byte, 64*1024)
	for {
		n, err := syscall.Read(fd, buf)
		if err != nil {
			if errors.Is(err, syscall.EINTR) {
				continue
			}
			return err
		}
		if n <= 0 {
			continue
		}
		w.consume(fd, buf[:n])
	}
}

func (w *Watcher) addTree(fd int, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.addWatch(fd, path)
		}
		return nil
	})
}

func (w *Watcher) addWatch(fd int, path string) error {
	mask := uint32(syscall.IN_CREATE | syscall.IN_MOVED_TO | syscall.IN_CLOSE_WRITE | syscall.IN_DELETE_SELF | syscall.IN_MOVE_SELF)
	wd, err := syscall.InotifyAddWatch(fd, path, mask)
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.wdToPath[wd] = path
	w.mu.Unlock()
	return nil
}

func (w *Watcher) consume(fd int, data []byte) {
	// The kernel inotify_event header is 16 bytes:
	// int32 wd, uint32 mask, uint32 cookie, uint32 len.
	// syscall.InotifyEvent has a zero-length Name field and unsafe.Sizeof can be
	// larger due to Go struct padding, so do not use unsafe.Sizeof here.
	const headerSize = 16
	for offset := 0; offset+headerSize <= len(data); {
		ev := syscall.InotifyEvent{
			Wd:     int32(nativeUint32(data[offset : offset+4])),
			Mask:   nativeUint32(data[offset+4 : offset+8]),
			Cookie: nativeUint32(data[offset+8 : offset+12]),
			Len:    nativeUint32(data[offset+12 : offset+16]),
		}
		offset += headerSize
		if offset+int(ev.Len) > len(data) {
			return
		}
		nameBytes := data[offset : offset+int(ev.Len)]
		offset += int(ev.Len)

		root := w.pathForWD(int(ev.Wd))
		if root == "" {
			continue
		}
		name := trimNull(nameBytes)
		path := root
		if name != "" {
			path = filepath.Join(root, name)
		}

		if ev.Mask&(syscall.IN_DELETE_SELF|syscall.IN_MOVE_SELF) != 0 {
			continue
		}
		if organizer.IsEphemeralPath(path) {
			continue
		}
		if ev.Mask&syscall.IN_ISDIR != 0 {
			if ev.Mask&(syscall.IN_CREATE|syscall.IN_MOVED_TO) != 0 {
				if err := w.addTree(fd, path); err != nil && !os.IsNotExist(err) {
					log.Printf("watch add %s: %v", path, err)
				}
			}
			continue
		}
		if ev.Mask&(syscall.IN_CLOSE_WRITE|syscall.IN_MOVED_TO|syscall.IN_CREATE) != 0 {
			w.schedule(path)
		}
	}
}

func (w *Watcher) pathForWD(wd int) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.wdToPath[wd]
}

func nativeUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func trimNull(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func (w *Watcher) schedule(path string) {
	if organizer.IsEphemeralPath(path) {
		return
	}
	clean := filepath.Clean(path)

	w.mu.Lock()
	defer w.mu.Unlock()
	if t, ok := w.timers[clean]; ok {
		t.Stop()
	}
	w.timers[clean] = time.AfterFunc(w.delay, func() {
		w.mu.Lock()
		delete(w.timers, clean)
		w.mu.Unlock()
		w.process(clean)
	})
}

func (w *Watcher) process(path string) {
	if organizer.IsEphemeralPath(path) {
		return
	}
	if err := waitStable(path, w.delay); err != nil {
		if organizer.IsIgnorableMissing(err) {
			return
		}
		log.Printf("stat %s: %v", path, err)
		return
	}
	if err := w.handle(path); err != nil {
		if organizer.IsIgnorableMissing(err) {
			return
		}
		log.Printf("handle %s: %v", path, err)
	}
}

func waitStable(path string, delay time.Duration) error {
	if delay <= 0 {
		delay = 1500 * time.Millisecond
	}
	interval := 250 * time.Millisecond
	maxWait := delay * 4
	if maxWait < 4*time.Second {
		maxWait = 4 * time.Second
	}
	if maxWait > 20*time.Second {
		maxWait = 20 * time.Second
	}

	deadline := time.Now().Add(maxWait)
	var lastSize int64 = -1
	var lastMod time.Time
	stable := 0

	for {
		st, err := os.Stat(path)
		if err != nil {
			return err
		}
		if st.IsDir() {
			return nil
		}
		if st.Size() == lastSize && st.ModTime().Equal(lastMod) {
			stable++
			if stable >= 2 {
				return nil
			}
		} else {
			stable = 0
			lastSize = st.Size()
			lastMod = st.ModTime()
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(interval)
	}
}
