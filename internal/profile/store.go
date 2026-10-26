package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/NaiKiDEV/ssh-chat/internal/chat"
)

const (
	fileVersion        = 1
	DefaultMaxProfiles = 10000
)

var (
	ErrStoreFull = errors.New("profile storage is full")
	ErrNoOwner   = errors.New("profile owner is required")
)

type fileFormat struct {
	Version  int                `json:"version"`
	Profiles map[string]Profile `json:"profiles"`
}

type Store struct {
	path        string
	maxProfiles int

	mu       sync.RWMutex
	profiles map[string]Profile
	owners   map[string]string
	version  uint64

	writeMu sync.Mutex
	written uint64
}

func NewMemoryStore() *Store {
	return newStore("")
}

func Open(path string) (*Store, error) {
	s := newStore(path)

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading profiles: %w", err)
	}

	var f fileFormat
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing profiles %s: %w", path, err)
	}
	if f.Version != fileVersion {
		return nil, fmt.Errorf("profiles %s: unsupported version %d", path, f.Version)
	}
	for owner, p := range f.Profiles {
		if err := s.load(owner, p); err != nil {
			return nil, fmt.Errorf("profiles %s: %w", path, err)
		}
	}
	return s, nil
}

func newStore(path string) *Store {
	return &Store{
		path:        path,
		maxProfiles: DefaultMaxProfiles,
		profiles:    make(map[string]Profile),
		owners:      make(map[string]string),
	}
}

func (s *Store) load(owner string, p Profile) error {
	if owner == "" {
		return ErrNoOwner
	}
	if err := p.Validate(); err != nil {
		return fmt.Errorf("profile %q: %w", owner, err)
	}
	key := chat.NameKey(p.Name)
	if other, taken := s.owners[key]; taken {
		return fmt.Errorf("name %q claimed by both %q and %q", p.Name, other, owner)
	}
	s.profiles[owner] = p
	s.owners[key] = owner
	return nil
}

func (s *Store) Get(owner string) (Profile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.profiles[owner]
	return p, ok
}

func (s *Store) NameOwner(nameKey string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	owner, ok := s.owners[nameKey]
	return owner, ok
}

func (s *Store) Save(owner string, p Profile) error {
	if owner == "" {
		return ErrNoOwner
	}
	if err := p.Validate(); err != nil {
		return err
	}

	prev, hadPrev, version, data, err := s.apply(owner, p)
	if err != nil {
		return err
	}
	if err := s.write(version, data); err != nil {
		s.rollback(owner, p, prev, hadPrev)
		return fmt.Errorf("saving profile: %w", err)
	}
	return nil
}

func (s *Store) apply(owner string, p Profile) (Profile, bool, uint64, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := chat.NameKey(p.Name)
	if other, taken := s.owners[key]; taken && other != owner {
		return Profile{}, false, 0, nil, chat.ErrNameTaken
	}
	prev, hadPrev := s.profiles[owner]
	if !hadPrev && len(s.profiles) >= s.maxProfiles {
		return Profile{}, false, 0, nil, ErrStoreFull
	}

	if hadPrev {
		delete(s.owners, chat.NameKey(prev.Name))
	}
	s.profiles[owner] = p
	s.owners[key] = owner
	s.version++

	if s.path == "" {
		return prev, hadPrev, s.version, nil, nil
	}
	data, err := json.MarshalIndent(fileFormat{Version: fileVersion, Profiles: s.profiles}, "", "  ")
	if err != nil {
		s.undoLocked(owner, p, prev, hadPrev)
		return Profile{}, false, 0, nil, fmt.Errorf("encoding profiles: %w", err)
	}
	return prev, hadPrev, s.version, data, nil
}

func (s *Store) rollback(owner string, p, prev Profile, hadPrev bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.undoLocked(owner, p, prev, hadPrev)
}

func (s *Store) undoLocked(owner string, p, prev Profile, hadPrev bool) {
	if cur, ok := s.profiles[owner]; !ok || cur != p {
		return
	}
	delete(s.owners, chat.NameKey(p.Name))
	delete(s.profiles, owner)
	if hadPrev {
		s.profiles[owner] = prev
		s.owners[chat.NameKey(prev.Name)] = owner
	}
}

func (s *Store) write(version uint64, data []byte) error {
	if s.path == "" {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if version <= s.written {
		return nil
	}
	if err := writeAtomic(s.path, data); err != nil {
		return err
	}
	s.written = version
	return nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".profiles-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
