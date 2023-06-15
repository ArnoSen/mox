package jmaphandler

const (
	contextUserKey = "_user"
)

//User is the object with userdata that is passed through context.
type User struct {
	Username string
}
