package chat

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultHistorySize  = 200
	DefaultMaxRooms     = 100
	DefaultMaxClients   = 200
	DefaultMessageRate  = rate.Limit(2)
	DefaultMessageBurst = 6

	MaxMessageLen   = 1000
	MaxMessageLines = 20
	maxColorLen     = 16
	maxNameSuffix   = 999
)

var (
	ErrClosed         = errors.New("server is shutting down")
	ErrServerFull     = errors.New("server is full, try again later")
	ErrTooManyRooms   = errors.New("too many rooms, join an existing one")
	ErrRateLimited    = errors.New("slow down a little")
	ErrEmptyMessage   = errors.New("message is empty")
	ErrMessageTooLong = fmt.Errorf("messages are limited to %d characters and %d lines", MaxMessageLen, MaxMessageLines)
	ErrNotInRoom      = errors.New("not in a room")
)

type Identity struct {
	ID    string
	Guest bool
}

type NameOwnerFunc func(nameKey string) (ownerID string, ok bool)

type Options struct {
	ListedRooms  []string
	HistorySize  int
	MaxRooms     int
	MaxClients   int
	MessageRate  rate.Limit
	MessageBurst int
	NameOwner    NameOwnerFunc
	Now          func() time.Time
}

type Hub struct {
	opts Options

	mu        sync.Mutex
	closed    bool
	nextID    uint64
	nextMsgID uint64
	listed    []string
	rooms     map[string]*room
	names     map[string]*Client
	clients   map[*Client]struct{}
}

type room struct {
	name     string
	listed   bool
	members  []*Client
	messages []Message
}

func NewHub(opts Options) (*Hub, error) {
	opts = withDefaults(opts)

	h := &Hub{
		opts:    opts,
		rooms:   make(map[string]*room),
		names:   make(map[string]*Client),
		clients: make(map[*Client]struct{}),
	}
	for _, raw := range opts.ListedRooms {
		name, err := NormalizeRoomName(raw)
		if err != nil {
			return nil, fmt.Errorf("listed room %q: %w", raw, err)
		}
		if _, dup := h.rooms[name]; dup {
			continue
		}
		h.rooms[name] = &room{name: name, listed: true}
		h.listed = append(h.listed, name)
	}
	if len(h.listed) > opts.MaxRooms {
		return nil, fmt.Errorf("%d listed rooms exceed the room limit of %d", len(h.listed), opts.MaxRooms)
	}
	return h, nil
}

func withDefaults(opts Options) Options {
	if opts.HistorySize <= 0 {
		opts.HistorySize = DefaultHistorySize
	}
	if opts.MaxRooms <= 0 {
		opts.MaxRooms = DefaultMaxRooms
	}
	if opts.MaxClients <= 0 {
		opts.MaxClients = DefaultMaxClients
	}
	if opts.MessageRate <= 0 {
		opts.MessageRate = DefaultMessageRate
	}
	if opts.MessageBurst <= 0 {
		opts.MessageBurst = DefaultMessageBurst
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return opts
}

func (h *Hub) Connect(id Identity, preferredName, color string) (*Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, ErrClosed
	}
	if len(h.clients) >= h.opts.MaxClients {
		return nil, ErrServerFull
	}
	name, ok := h.pickNameLocked(preferredName, id)
	if !ok {
		return nil, ErrNameTaken
	}

	h.nextID++
	c := &Client{
		hub:      h,
		id:       h.nextID,
		identity: id,
		updates:  make(chan struct{}, 1),
		done:     make(chan struct{}),
		limiter:  rate.NewLimiter(h.opts.MessageRate, h.opts.MessageBurst),
		name:     name,
		color:    cleanColor(color),
	}
	h.clients[c] = struct{}{}
	h.names[NameKey(name)] = c
	return c, nil
}

func (h *Hub) Rooms() []RoomInfo {
	h.mu.Lock()
	defer h.mu.Unlock()

	out := make([]RoomInfo, 0, len(h.listed))
	for _, name := range h.listed {
		out = append(out, RoomInfo{Name: name, Online: len(h.rooms[name].members)})
	}
	return out
}

func (h *Hub) Online() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) Closed() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closed
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.closed = true
	for c := range h.clients {
		h.disconnectLocked(c)
	}
}

func (h *Hub) pickNameLocked(preferred string, id Identity) (string, bool) {
	base := nameCandidate(preferred)
	if base == "" {
		base = "guest"
	}
	if h.nameAvailableLocked(base, id, nil) {
		return base, true
	}
	for i := 2; i <= maxNameSuffix; i++ {
		suffix := strconv.Itoa(i)
		candidate := truncateRunes(base, MaxNameLen-len(suffix)) + suffix
		if h.nameAvailableLocked(candidate, id, nil) {
			return candidate, true
		}
	}
	return "", false
}

func (h *Hub) nameAvailableLocked(name string, id Identity, self *Client) bool {
	key := NameKey(name)
	if isReserved(key) {
		return false
	}
	if holder, ok := h.names[key]; ok && holder != self {
		return false
	}
	if h.opts.NameOwner != nil {
		if owner, ok := h.opts.NameOwner(key); ok && (id.Guest || owner != id.ID) {
			return false
		}
	}
	return true
}

func (h *Hub) disconnectLocked(c *Client) {
	if c.closed {
		return
	}
	h.leaveLocked(c)
	c.closed = true
	delete(h.clients, c)
	if h.names[NameKey(c.name)] == c {
		delete(h.names, NameKey(c.name))
	}
	close(c.done)
}

func (h *Hub) leaveLocked(c *Client) {
	r := c.room
	if r == nil {
		return
	}
	r.members = slices.DeleteFunc(r.members, func(m *Client) bool { return m == c })
	c.room = nil

	if len(r.members) == 0 && !r.listed {
		delete(h.rooms, r.name)
	} else {
		h.appendLocked(r, Message{Kind: KindLeave, Author: c.memberLocked()})
	}
	if r.listed {
		h.notifyLobbyLocked()
	}
}

func (h *Hub) appendLocked(r *room, msg Message) {
	h.nextMsgID++
	msg.ID = h.nextMsgID
	msg.Time = h.opts.Now()

	r.messages = append(r.messages, msg)
	if over := len(r.messages) - h.opts.HistorySize; over > 0 {
		r.messages = slices.Delete(r.messages, 0, over)
	}
	h.notifyRoomLocked(r)
}

func (h *Hub) notifyRoomLocked(r *room) {
	for _, m := range r.members {
		m.notify()
	}
}

func (h *Hub) notifyLobbyLocked() {
	for c := range h.clients {
		if c.room == nil {
			c.notify()
		}
	}
}

func cleanColor(color string) string {
	return truncateRunes(nameCandidate(color), maxColorLen)
}
