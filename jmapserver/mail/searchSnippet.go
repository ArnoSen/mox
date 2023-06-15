package mail

type SearchSnippet struct {
}

func NewSearchSnippet() SearchSnippet {
	return SearchSnippet{}
}

func (m SearchSnippet) Name() string {
	return "SearchSnippet"
}
