package chat

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/NaiKiDEV/ssh-chat/internal/sanitize"
	"golang.org/x/time/rate"
)

type Client struct {
	hub      *Hub
	id       uint64
	identity Identity
	updates  chan struct{}
	done     chan struct{}
	limiter  *rate.Limiter

	name   string
	color  string
	room   *room
	closed bool
}

func (c *Client) ID() uint64 { return c.id }

func (c *Client) Identity() Identity { return c.identity }

func (c *Client) Updates() <-chan struct{} { return c.updates }

func (c *Client) Done() <-chan struct{} { return c.done }

func (c *Client) Member() Member {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	return c.memberLocked()
}

func (c *Client) Rooms() []RoomInfo { return c.hub.Rooms() }

func (c *Client) Join(name string) (string, error) {
	roomName, err := NormalizeRoomName(name)
	if err != nil {
		return "", err
	}

	h := c.hub
	h.mu.Lock()
	defer h.mu.Unlock()

	if c.closed {
		return "", ErrClosed
	}
	if c.room != nil && c.room.name == roomName {
		return roomName, nil
	}
	if !c.limiter.Allow() {
		return "", ErrRateLimited
	}
	r, ok := h.rooms[roomName]
	if !ok {
		if len(h.rooms) >= h.opts.MaxRooms {
			return "", ErrTooManyRooms
		}
		r = &room{name: roomName}
		h.rooms[roomName] = r
	}

	h.leaveLocked(c)
	r.members = append(r.members, c)
	c.room = r
	h.appendLocked(r, Message{Kind: KindJoin, Author: c.memberLocked()})
	if r.listed {
		h.notifyLobbyLocked()
	}
	return roomName, nil
}

func (c *Client) Leave() {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	c.hub.leaveLocked(c)
}

func (c *Client) Send(text string) error { return c.post(KindChat, text) }

func (c *Client) Emote(text string) error { return c.post(KindEmote, text) }

func (c *Client) post(kind Kind, text string) error {
	body, err := cleanMessage(text)
	if err != nil {
		return err
	}

	h := c.hub
	h.mu.Lock()
	defer h.mu.Unlock()

	if c.closed {
		return ErrClosed
	}
	if c.room == nil {
		return ErrNotInRoom
	}
	if !c.limiter.Allow() {
		return ErrRateLimited
	}
	h.appendLocked(c.room, Message{Kind: kind, Author: c.memberLocked(), Text: body})
	return nil
}

func (c *Client) Rename(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	h := c.hub
	h.mu.Lock()
	defer h.mu.Unlock()

	if c.closed {
		return ErrClosed
	}
	if name == c.name {
		return nil
	}
	if !h.nameAvailableLocked(name, c.identity, c) {
		return ErrNameTaken
	}
	if !c.limiter.Allow() {
		return ErrRateLimited
	}

	old := c.name
	delete(h.names, NameKey(old))
	h.names[NameKey(name)] = c
	c.name = name
	if c.room != nil {
		h.appendLocked(c.room, Message{Kind: KindRename, Author: c.memberLocked(), Text: old})
	}
	return nil
}

func (c *Client) SetColor(color string) {
	h := c.hub
	h.mu.Lock()
	defer h.mu.Unlock()

	c.color = cleanColor(color)
	if c.room != nil {
		h.notifyRoomLocked(c.room)
	}
}

func (c *Client) Room() (RoomSnapshot, bool) {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()

	r := c.room
	if r == nil {
		return RoomSnapshot{}, false
	}
	members := make([]Member, len(r.members))
	for i, m := range r.members {
		members[i] = m.memberLocked()
	}
	slices.SortFunc(members, func(a, b Member) int {
		return strings.Compare(NameKey(a.Name), NameKey(b.Name))
	})
	return RoomSnapshot{
		Name:     r.name,
		Listed:   r.listed,
		Members:  members,
		Messages: slices.Clone(r.messages),
	}, true
}

func (c *Client) Close() {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	c.hub.disconnectLocked(c)
}

func (c *Client) memberLocked() Member {
	return Member{ID: c.id, Name: c.name, Color: c.color, Guest: c.identity.Guest}
}

func (c *Client) notify() {
	select {
	case c.updates <- struct{}{}:
	default:
	}
}

func cleanMessage(text string) (string, error) {
	body := sanitize.Text(text)
	body = strings.TrimRight(strings.TrimLeft(body, "\n"), " \n")
	if strings.TrimSpace(body) == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > MaxMessageLen || strings.Count(body, "\n")+1 > MaxMessageLines {
		return "", ErrMessageTooLong
	}
	return body, nil
}
