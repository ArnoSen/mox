package jaccount

import (
	"context"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/message"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

var validEmailFilters []string = []string{
	"inMailbox",
	"inMailboxOtherThan",
	"before",
	"after",
	"minSize",
	"maxSize",
	"allInThreadHaveKeyword",
	"someInThreadHaveKeyword",
	"noneInThreadHaveKeyword",
	"hasKeyword",
	"notKeyword",
	"hasAttachment",
	"text",
	"from",
	"to",
	"cc",
	"bcc",
	"subject",
	"body",
	"header",
}

var validSortProperties []string = []string{
	"receivedAt",
	"size",
	"from",
	"to",
	"subject",
	"sentAt",
	"hasKeyword",
	"allInThreadHaveKeyword",
	"someInThreadHaveKeyword",
}

type Email struct {
	EmailMetadata          //4.1.1
	HeaderFieldParsedForms //4.1.2
	HeaderFieldsProperties //4.1.3
	EmailBodyParts         //4.1.4

}

type EmailBodyParts struct {
	BodyStructure EmailBodyPart             `json:"bodyStructure"`
	BodyValues    map[string]EmailBodyValue `json:"bodyValues"`
	TextBody      []EmailBodyPart           `json:"textBody"`
	HTMLBody      []EmailBodyPart           `json:"htmlBody"`
	Attachments   []EmailBodyPart           `json:"attachments"`
	HasAttachment bool                      `json:"hasAttachment"`
	Preview       string                    `json:"preview"`
}

type HeaderFieldsProperties struct {
	Headers    []EmailHeader   `json:"headers"`
	MessageId  []string        `json:"messageId"`
	InReplyTo  []string        `json:"inReplyTo"`
	References []string        `json:"references"`
	Sender     []EmailAddress  `json:"sender"`
	From       []EmailAddress  `json:"from"`
	To         []EmailAddress  `json:"to"`
	CC         []EmailAddress  `json:"cc"`
	BCC        []EmailAddress  `json:"bcc"`
	ReplyTo    []EmailAddress  `json:"replyTo"`
	Subject    *string         `json:"subject"`
	SentAt     *basetypes.Date `json:"sentAt"`
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

func (ja *JAccount) QueryEmail(ctx context.Context, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit int, calculateTotal bool, collapseThreads bool) (queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {

	ja.mlog.Debug("JAccount QueryEmail", mlog.Field("collapseThreads", collapseThreads))

	//FIXME implement collapseThreads

	q := bstore.QueryDB[store.Message](ctx, ja.mAccount.DB)

	q2 := bstore.QueryDB[store.Message](ctx, ja.mAccount.DB)

	//qTotal is a secondary query that we may need to calculate the total
	//var qTotal bstore.QueryDB[store.Message]

	if filter != nil {
		filterCondition, ok := filter.GetFilter().(basetypes.FilterCondition)
		if !ok {
			//let's do only simple filters for now
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("only filterconditions are supported for now")
		}

		switch filterCondition.Property {
		case "inMailbox":
			var mailboxIDint int64
			switch filterCondition.AssertedValue.(type) {
			case int:
				mailboxIDint = int64(filterCondition.AssertedValue.(int))
			case string:
				var parseErr error
				mailboxIDint, parseErr = strconv.ParseInt(filterCondition.AssertedValue.(string), 10, 64)
				if parseErr != nil {
					return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("inMailbox filter value must be a (quoted) integer")
				}
			default:
				return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("inMailbox filter value must be a (quoted) integer")
			}

			q.FilterNonzero(store.Message{
				MailboxID: int64(mailboxIDint),
			})

			q2.FilterNonzero(store.Message{
				MailboxID: int64(mailboxIDint),
			})

		default:
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("unsupported filter")
		}

	}

	for _, s := range sort {
		switch s.Property {
		case "receivedAt":
			if s.IsAscending {
				q.SortAsc("Received")
				q2.SortAsc("Received")
			}
			q.SortDesc("Received")
			q2.SortDesc("Received")
		default:
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedSort("unsupported sort parameter")
		}
	}

	q.Limit(int(position) + limit)

	if calculateTotal {
		//TODO looking at the implementation of Count, maybe it is better we calc the total in the next for loop
		q2.Limit(math.MaxInt)
		totalCnt, countErr := q2.Count()
		if countErr != nil {
			ja.mlog.Error("error getting count", mlog.Field("err", countErr.Error()))
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorServerFail()
		}
		total = basetypes.Uint(totalCnt)
	}

	var (
		//FIXME position can also be negative. In that case results need to come from the other end of the list.
		skip      int64 = int64(position)
		i         int64
		threadMap map[int64]interface{} = make(map[int64]interface{})
	)

	for {
		i++
		if i-1 < skip {
			continue
		}

		if !collapseThreads {
			var id int64
			if err := q.NextID(&id); err == bstore.ErrAbsent {
				// No more messages.
				// Note: if we don't iterate until an error, Close must be called on the query for cleanup.
				break
			} else if err != nil {
				ja.mlog.Error("error getting next id", mlog.Field("err", err.Error()))
				return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorServerFail()
			}

			// The ID is fetched from the index. The full record is
			// never read from the database. Calling Next instead
			// of NextID does always fetch, parse and return the
			// full record.
			ids = append(ids, basetypes.NewIdFromInt64(id))
		} else {
			msg, err := q.Next()
			if err == bstore.ErrAbsent {
				break
			} else if err != nil {
				ja.mlog.Error("error getting message", mlog.Field("err", err.Error()))
				return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorServerFail()
			}

			if _, ok := threadMap[msg.ThreadID]; !ok {
				ids = append(ids, basetypes.NewIdFromInt64(msg.ID))
				threadMap[msg.ThreadID] = nil
			}
		}
	}

	return "stubstate", false, position, ids, total, nil
}

func (ja *JAccount) GetEmail(ctx context.Context, ids []basetypes.Id, properties []string, bodyProperties []string, FetchTextBodyValues, FetchHTMLBodyValues, FetchAllBodyValues bool, MaxBodyValueBytes basetypes.Uint) (state string, result []Email, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	//TODO:
	// implement properties:  blobId, inReplyTo, references, header:list-id:asText, header:list-post:asURLs, replyTo, sentAt, bodyStructure, bodyValues
	// implement body parameters: partId, blobId, size, name, type, charset, disposition, cid, location

	for _, id := range ids {
		idInt64, err := id.Int64()
		if err != nil {
			//the email ids are imap ids meaning they are int64
			notFound = append(notFound, id)
			continue
		}

		em := store.Message{
			ID: idInt64,
		}

		if err := ja.mAccount.DB.Get(ctx, &em); err != nil {
			if err == bstore.ErrAbsent {
				notFound = append(notFound, id)
				continue
			}
			ja.mlog.Error("error getting message from db", mlog.Field("id", idInt64), mlog.Field("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement := Email{
			EmailMetadata: EmailMetadata{
				Id:       id,
				ThreadId: basetypes.NewIdFromInt64(em.ThreadID),
				MailboxIds: map[basetypes.Id]bool{
					basetypes.NewIdFromInt64(em.MailboxID): true,
				},
				Size:       basetypes.Uint(em.Size),
				ReceivedAt: basetypes.UTCDate(em.Received),
				Keywords:   flagsToKeywords(em.Flags),
			},
			EmailBodyParts: EmailBodyParts{
				Preview: "<preview not available>",
			},
		}

		if HasAny(properties, "from", "subject", "to", "messageId", "date", "sender", "preview") {
			part, err := em.LoadPart(ja.mAccount.MessageReader(em))
			if err != nil {
				ja.mlog.Error("error load message part", mlog.Field("id", idInt64), mlog.Field("error", err.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}

			ja.mlog.Debug("dump part", mlog.Field("var", part.VarString()))

			if env := part.Envelope; env != nil {
				resultElement.HeaderFieldsProperties.MessageId = []string{env.MessageID}
				d := basetypes.Date(env.Date)
				resultElement.HeaderFieldParsedForms.Date = &d

				if part.Envelope.Subject != "" {
					resultElement.HeaderFieldsProperties.Subject = &env.Subject
				}
				for _, from := range env.From {
					resultElement.HeaderFieldsProperties.From = append(resultElement.HeaderFieldsProperties.From, msgAddressToEmailAddress(from))
				}
				for _, to := range env.To {
					resultElement.HeaderFieldsProperties.To = append(resultElement.HeaderFieldsProperties.To, msgAddressToEmailAddress(to))
				}
				for _, cc := range env.CC {
					resultElement.HeaderFieldsProperties.CC = append(resultElement.HeaderFieldsProperties.CC, msgAddressToEmailAddress(cc))
				}
				for _, bcc := range env.BCC {
					resultElement.HeaderFieldsProperties.BCC = append(resultElement.HeaderFieldsProperties.BCC, msgAddressToEmailAddress(bcc))
				}
				for _, sender := range env.Sender {
					resultElement.HeaderFieldsProperties.Sender = append(resultElement.HeaderFieldsProperties.Sender, msgAddressToEmailAddress(sender))
				}
			}

			//read the whole body and see what we got
			fullBody, err := io.ReadAll(part.Reader())
			if err != nil {
				ja.mlog.Error("error loading body", mlog.Field("id", idInt64), mlog.Field("error", err.Error()))
			} else if len(fullBody) < 100 {
				resultElement.EmailBodyParts.Preview = string(fullBody)
			} else {
				resultElement.EmailBodyParts.Preview = string(fullBody[:100])
			}
		}

		result = append(result, resultElement)
	}

	return "stubstate", result, notFound, nil
}

func msgAddressToEmailAddress(m message.Address) EmailAddress {
	result := EmailAddress{
		Email: fmt.Sprintf("%s@%s", m.User, m.Host),
	}
	if m.Name != "" {
		result.Name = &m.Name
	}
	return result
}

// HasAny returns true haystack has any needles
func HasAny(haystack []string, needle ...string) bool {
	for _, h := range haystack {
		for _, n := range needle {
			if h == n {
				return true
			}
		}
	}
	return false
}

func flagsToKeywords(f store.Flags) map[string]bool {
	result := make(map[string]bool)
	if f.Answered {
		result["$answered"] = true
	}
	if f.Deleted {
		//FIXME need to make sure in all operations that this is guaranteed
		//Any message with the \Deleted keyword MUST NOT be visible via JMAP
	}
	if f.Draft {
		result["$draft"] = true
	}
	if f.Flagged {
		result["$flagged"] = true
	}
	if f.Forwarded {
		result["$forwarded"] = true
	}
	if f.Junk {
		result["$junk"] = true
	}
	if f.MDNSent {
	}
	if f.Notjunk {
		result["$notjunk"] = true
	}
	if f.Phishing {
		result["$phishing"] = true
	}
	if f.Seen {
		result["$seen"] = true
	}
	return result
}
