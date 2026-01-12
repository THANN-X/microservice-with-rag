package domain

import "time"

type InboxEvent struct {
	ID          string
	ConsumerID  string
	ProcessedAt time.Time
}
