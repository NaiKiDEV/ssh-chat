package chat

import "time"

type Kind int

const (
	KindChat Kind = iota
	KindEmote
	KindJoin
	KindLeave
	KindRename // Text holds the previous name
)

type Member struct {
	ID    uint64
	Name  string
	Color string
	Guest bool
}

type Message struct {
	ID     uint64
	Kind   Kind
	Author Member
	Text   string
	Time   time.Time
}

type RoomInfo struct {
	Name   string
	Online int
}

type RoomSnapshot struct {
	Name     string
	Listed   bool
	Members  []Member
	Messages []Message
}
