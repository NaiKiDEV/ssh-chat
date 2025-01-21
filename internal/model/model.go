package model

import "time"

type Message struct {
	Username  string
	Text      string
	Timestamp time.Time
}
