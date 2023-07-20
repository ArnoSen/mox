package mailcapability

import "github.com/mjl-/mox/jmapserver/basetypes"

type ThreadDT struct {
}

func NewThread() ThreadDT {
	return ThreadDT{}
}

func (t ThreadDT) Name() string {
	return "Thread"
}

type Thread struct {
	Id       basetypes.Id   `json:"id"`
	EmailIds []basetypes.Id `json:"emailIds"`
}
