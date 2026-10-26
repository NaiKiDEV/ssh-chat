package chat

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/NaiKiDEV/ssh-chat/internal/sanitize"
)

const (
	MaxNameLen     = 20
	MaxRoomNameLen = 24
)

var reservedNames = []string{
	"admin", "administrator", "mod", "moderator", "official", "root",
	"server", "ssh-chat", "staff", "support", "sysop", "system",
}

var (
	ErrInvalidName = errors.New("names use a-z, A-Z, 0-9, '-', '_' or '.' and start with a letter or digit (max 20)")
	ErrNameTaken   = errors.New("that name is already taken")
	ErrInvalidRoom = errors.New("room names use a-z, 0-9, '-' or '_' (max 24)")
)

func ValidateName(name string) error {
	if n := utf8.RuneCountInString(name); n == 0 || n > MaxNameLen {
		return ErrInvalidName
	}
	for i, r := range name {
		if !isNameRune(r) || (i == 0 && !isNameStart(r)) {
			return ErrInvalidName
		}
	}
	return nil
}

func NameKey(name string) string {
	return strings.ToLower(name)
}

func NormalizeRoomName(name string) (string, error) {
	name = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "#"))
	if name == "" || len(name) > MaxRoomNameLen {
		return "", ErrInvalidRoom
	}
	for _, r := range name {
		if !isRoomRune(r) {
			return "", ErrInvalidRoom
		}
	}
	return name, nil
}

func nameCandidate(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range sanitize.Line(s) {
		if n == MaxNameLen {
			break
		}
		if !isNameRune(r) || (n == 0 && !isNameStart(r)) {
			continue
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func isReserved(nameKey string) bool {
	return slices.Contains(reservedNames, nameKey)
}

func isNameStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isNameRune(r rune) bool {
	return isNameStart(r) || r == '-' || r == '_' || r == '.'
}

func isRoomRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}
