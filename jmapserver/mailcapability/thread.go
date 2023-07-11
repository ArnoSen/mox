package mailcapability

import "github.com/mjl-/mox/jmapserver/datatyper"

type ThreadDT struct {
}

func NewThread() ThreadDT {
	return ThreadDT{}
}

func (t ThreadDT) Name() string {
	return "Thread"
}

type Thread struct {
	Id       datatyper.Id   `json:"id"`
	EmailIds []datatyper.Id `json:"emailIds"`
}
