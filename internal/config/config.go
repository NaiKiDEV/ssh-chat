package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
)

const (
	envPrefix      = "SSH_CHAT_"
	maxHistorySize = 10000
)

type Config struct {
	Host         string
	Port         int
	HostKeyPath  string
	ProfilesPath string
	Rooms        []string
	HistorySize  int
	MaxRooms     int
	MaxUsers     int
	AllowGuests  bool
	IdleTimeout  time.Duration
	LogLevel     log.Level
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func Load(args []string, getenv func(string) string) (Config, error) {
	cfg := Config{
		Rooms:    []string{"public"},
		LogLevel: log.InfoLevel,
	}

	fs := flag.NewFlagSet("ssh-chat", flag.ContinueOnError)
	fs.StringVar(&cfg.Host, "host", "localhost", "address to listen on")
	fs.IntVar(&cfg.Port, "port", 23234, "port to listen on")
	fs.StringVar(&cfg.HostKeyPath, "host-key", ".ssh/id_ed25519", "host key path, generated when missing")
	fs.StringVar(&cfg.ProfilesPath, "profiles", ".data/profiles.json", "profile storage file, empty keeps profiles in memory")
	fs.Func("rooms", "comma separated rooms listed in the lobby (default \"public\")", func(v string) error {
		cfg.Rooms = splitList(v)
		return nil
	})
	fs.IntVar(&cfg.HistorySize, "history", chat.DefaultHistorySize, "messages kept per room")
	fs.IntVar(&cfg.MaxRooms, "max-rooms", chat.DefaultMaxRooms, "maximum number of rooms")
	fs.IntVar(&cfg.MaxUsers, "max-users", chat.DefaultMaxClients, "maximum concurrent connections")
	fs.BoolVar(&cfg.AllowGuests, "allow-guests", true, "allow connections without an SSH key")
	fs.DurationVar(&cfg.IdleTimeout, "idle-timeout", time.Hour, "drop connections idle this long, 0 disables")
	fs.Func("log-level", "debug, info, warn or error (default \"info\")", func(v string) error {
		lvl, err := log.ParseLevel(v)
		cfg.LogLevel = lvl
		return err
	})

	if err := applyEnv(fs, getenv); err != nil {
		return Config{}, err
	}
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyEnv(fs *flag.FlagSet, getenv func(string) string) error {
	var errs []error
	fs.VisitAll(func(f *flag.Flag) {
		name := envPrefix + strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		if v := getenv(name); v != "" {
			if err := fs.Set(f.Name, v); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
		}
	})
	return errors.Join(errs...)
}

func (c Config) validate() error {
	var errs []error
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("port %d is out of range", c.Port))
	}
	if c.HostKeyPath == "" {
		errs = append(errs, errors.New("host-key is required"))
	}
	if c.HistorySize < 1 || c.HistorySize > maxHistorySize {
		errs = append(errs, fmt.Errorf("history must be between 1 and %d", maxHistorySize))
	}
	if c.MaxRooms < max(len(c.Rooms), 1) {
		errs = append(errs, fmt.Errorf("max-rooms must be at least %d", max(len(c.Rooms), 1)))
	}
	if c.MaxUsers < 1 {
		errs = append(errs, errors.New("max-users must be at least 1"))
	}
	if c.IdleTimeout < 0 {
		errs = append(errs, errors.New("idle-timeout cannot be negative"))
	}
	for _, r := range c.Rooms {
		if _, err := chat.NormalizeRoomName(r); err != nil {
			errs = append(errs, fmt.Errorf("room %q: %w", r, err))
		}
	}
	return errors.Join(errs...)
}

func splitList(v string) []string {
	var out []string
	for part := range strings.SplitSeq(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
