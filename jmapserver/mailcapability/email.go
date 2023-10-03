package mailcapability

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type EmailDT struct {
	//maxQueryLimit is the number of emails returned for a query request
	maxQueryLimit int
}

func NewEmail(maxQueryLimit int) EmailDT {
	return EmailDT{
		maxQueryLimit: maxQueryLimit,
	}
}

func (m EmailDT) Name() string {
	return "Email"
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.5
func (m EmailDT) Query(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit *basetypes.Uint, calculateTotal bool) (retAccountId basetypes.Id, queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, retLimit basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {

	//FIXME
	//Need to handle collapseThreads ../../rfc/8621:2506

	var adjustedLimit int = m.maxQueryLimit

	if limit != nil && int(*limit) < adjustedLimit {
		adjustedLimit = int(*limit)
	}

	state, canCalculateChanges, retPosition, ids, total, mErr := jaccount.QueryEmail(ctx, filter, sort, position, anchor, anchorOffset, adjustedLimit, calculateTotal)

	return accountId, state, canCalculateChanges, basetypes.Int(retPosition), ids, total, basetypes.Uint(adjustedLimit), mErr
}

type Email struct {
	TextBody               string
	HTMLBody               string
	EmailMetadata          //4.1.1
	HeaderFieldParsedForms //4.1.2
	HeaderFieldsProperties //4.1.3
	EmailBodyPart          //4.1.4
}

type HeaderFieldsProperties struct {
	Headers       []EmailHeader             `json:"headers"`
	MessageId     []string                  `json:"messageId"`
	InReplyTo     []string                  `json:"inReplyTo"`
	References    []string                  `json:"references"`
	Sender        []EmailAddress            `json:"sender"`
	From          []EmailAddress            `json:"from"`
	To            []EmailAddress            `json:"to"`
	CC            []EmailAddress            `json:"cc"`
	BCC           []EmailAddress            `json:"bcc"`
	ReplyTo       []EmailAddress            `json:"replyTo"`
	Subject       *string                   `json:"subject"`
	SentAt        *basetypes.Date           `json:"sentAt"`
	BodyStructure EmailBodyPart             `json:"bodyStructure"`
	BodyValues    map[string]EmailBodyValue `json:"bodyValues"`
	TextBody      []EmailBodyPart           `json:"textBody"`
	HTMLBody      []EmailBodyPart           `json:"htmlBody"`
	Attachments   []EmailBodyPart           `json:"attachments"`
	HasAttachment bool                      `json:"hasAttachment"`
	Preview       string                    `json:"preview"`
}

type EmailBodyValue struct {
	Value             string `json:"value"`
	IsEncodingProblem bool   `json:"isEncodingProblem"`
	IsTruncated       bool   `json:"isTruncated"`
}

type EmailMetadata struct {
	Id         basetypes.Id          `json:"id"`
	BlobId     basetypes.Id          `json:"blobId"`
	ThreadId   basetypes.Id          `json:"threadId"`
	MailboxIds map[basetypes.Id]bool `json:"mailboxIds"`
	Keywords   map[string]bool       `json:"keywords"`
	Size       basetypes.Uint        `json:"size"`
	ReceivedAt basetypes.UTCDate     `json:"receivedAt"`
}

type HeaderFieldParsedForms struct {
	Raw            string              `json:"raw"`
	Text           string              `json:"text"`
	Addresses      []EmailAddress      `json:"addresses"`
	GroupAddresses []EmailAddressGroup `json:"groupAddresses"`
	MessageIds     []string            `json:"messageIds"`
	Date           *basetypes.Date     `json:"date"`
	URLs           []string            `json:"urls"`
}

type EmailAddress struct {
	Name  *string `json:"name"`
	Email string  `json:"email"`
}

type EmailAddressGroup struct {
	Name      *string        `json:"name"`
	Addresses []EmailAddress `json:"addresses"`
}

type EmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type EmailBodyPart struct {
	PartId      *string         `json:"partId"`
	BlobId      *basetypes.Id   `json:"blobId"`
	Size        basetypes.Uint  `json:"size"`
	Headers     []EmailHeader   `json:"headers"`
	Name        *string         `json:"name"`
	Type        *string         `json:"type"`
	CharSet     *string         `json:"charSet"`
	Disposition *string         `json:"disposition"`
	Cid         *string         `json:"cid"`
	Language    *string         `json:"language"`
	SubParts    []EmailBodyPart `json:"subParts"`
}
