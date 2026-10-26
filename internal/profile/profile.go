package profile

import (
	"errors"
	"hash/fnv"
	"regexp"
	"slices"
	"time"

	"github.com/NaiKiDEV/ssh-chat/internal/chat"
)

var Colors = []string{"orange", "yellow", "lime", "green", "teal", "sky", "blue", "purple"}

var (
	ErrInvalidColor    = errors.New("unknown color")
	ErrInvalidTimeZone = errors.New("unknown time zone, use a name like Europe/Vilnius or UTC")
)

var timeZonePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-]*(/[A-Za-z0-9_+\-]+)*$`)

const maxTimeZoneLen = 64

type Profile struct {
	Name     string `json:"name"`
	Color    string `json:"color,omitempty"`
	TimeZone string `json:"timezone,omitempty"`
}

func (p Profile) Validate() error {
	if err := chat.ValidateName(p.Name); err != nil {
		return err
	}
	if p.Color != "" && !slices.Contains(Colors, p.Color) {
		return ErrInvalidColor
	}
	if _, err := LoadLocation(p.TimeZone); err != nil {
		return err
	}
	return nil
}

func (p Profile) Location() *time.Location {
	loc, err := LoadLocation(p.TimeZone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func LoadLocation(name string) (*time.Location, error) {
	if name == "" {
		return time.UTC, nil
	}
	if len(name) > maxTimeZoneLen || !timeZonePattern.MatchString(name) {
		return nil, ErrInvalidTimeZone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, ErrInvalidTimeZone
	}
	return loc, nil
}

func DefaultColor(seed string) string {
	h := fnv.New32a()
	h.Write([]byte(seed))
	return Colors[h.Sum32()%uint32(len(Colors))]
}
