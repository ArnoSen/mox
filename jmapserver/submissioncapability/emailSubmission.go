package submission

type EmailSubmission struct {
}

func NewEmailSubmission() EmailSubmission {
	return EmailSubmission{}
}

func (m EmailSubmission) Name() string {
	return "EmailSubmission"
}
