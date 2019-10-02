package models

type EventType int

const (
	EVENT_JOIN = iota
	EVENT_LEAVE
	EVENT_MESSAGE
)

const (
	ECHO  = "echo"
	NORTH = "north"
)

type Event struct {
	Type      EventType // JOIN, LEAVE, MESSAGE
	Addr      string
	Timestamp int // Unix timestamp (secs)
	Content   string
}
