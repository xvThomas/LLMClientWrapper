package internal

// MessageStore persists conversation messages.
type MessageStore interface {
	Add(msg Message)
	All() []Message
	Clear()
}
