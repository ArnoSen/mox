package jmaphandler

const (
	defaultContextUserKey = "_user"
)

// User is the object with userdata that is passed through context.
type User struct {
	Username string
}
