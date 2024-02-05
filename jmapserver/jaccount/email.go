package jaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/mail"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/exp/slog"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/message"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

const previewNotAvailableText = "<preview not available>"

var validEmailFilters []string = []string{
	"inMailbox", "inMailboxOtherThan", "before", "after", "minSize",
	"maxSize", "allInThreadHaveKeyword", "someInThreadHaveKeyword", "noneInThreadHaveKeyword",
	"hasKeyword", "notKeyword", "hasAttachment", "text",
	"from", "to", "cc", "bcc",
	"subject", "body", "header",
}

var validSortProperties []string = []string{
	"receivedAt", "size", "from", "to",
	"subject", "sentAt", "hasKeyword", "allInThreadHaveKeyword",
	"someInThreadHaveKeyword",
}

var defaultEmailPropertyFields = []string{
	"id", "blobId", "threadId", "mailboxIds", "keywords", "size",
	"receivedAt", "messageId", "inReplyTo", "references", "sender", "from",
	"to", "cc", "bcc", "replyTo", "subject", "sentAt", "hasAttachment",
	"preview", "bodyValues", "textBody", "htmlBody", "attachments",
}

var defaultEmailBodyProperties = []string{
	"partId", "blobId", "size", "name", "type", "charset",
	"disposition", "cid", "language", "location",
}

type EmailDefinedProperties struct {
	EmailMetadata          //4.1.1
	HeaderFieldsProperties //4.1.3
	EmailBodyParts         //4.1.4
}

type Email struct {
	EmailDefinedProperties
	DynamicProperties map[string]any `json:"-"` // we need a custom marshaller for this

	//properties is used in MarshalJSON to filter the fields we need
	properties []string
}

// Marshal is a custom marshaler that is needed to get requested properties in the result that are known only at runtime. Examples of custom properties are e.g. headers that the client is interested in. Also it limits the output to the properties that we need. This is done for performance reasons (otherwise we keep (un) marhalling all the time)
func (e Email) MarshalJSON() ([]byte, error) {
	//there must be a simpeler method than this
	emailBytes, err := json.Marshal(e.EmailDefinedProperties)
	if err != nil {
		return nil, err
	}

	var emailMapStringAny = make(map[string]any, 0)

	if err := json.Unmarshal(emailBytes, &emailMapStringAny); err != nil {
		return nil, err
	}

	e.properties = append(e.properties, "id")

	//remove all the fields we do not need exepct for 'id'
	for k := range emailMapStringAny {
		var keepProperty bool
		for _, p := range e.properties {
			if k == p {
				keepProperty = true
				break
			}
		}
		if !keepProperty {
			delete(emailMapStringAny, k)
		}
	}

	for k, v := range e.DynamicProperties {
		emailMapStringAny[k] = v
	}
	return json.Marshal(emailMapStringAny)
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
	Headers    []EmailHeader  `json:"headers"`
	MessageId  []string       `json:"messageId"`
	InReplyTo  []string       `json:"inReplyTo"`
	References []string       `json:"references"`
	Sender     []EmailAddress `json:"sender"`
	From       []EmailAddress `json:"from"`
	To         []EmailAddress `json:"to"`
	CC         []EmailAddress `json:"cc"`
	BCC        []EmailAddress `json:"bcc"`
	ReplyTo    []EmailAddress `json:"replyTo"`

	//The value is identical to the value of header:Subject:asText.
	Subject *string         `json:"subject"`
	SentAt  *basetypes.Date `json:"sentAt"`
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

type EmailAddress struct {
	Name  *string `json:"name"`
	Email string  `json:"email"`
}

// EmailAddressGroup is not yet implemented in mox
type EmailAddressGroup struct {
	Name      *string        `json:"name"`
	Addresses []EmailAddress `json:"addresses"`
}

type EmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type EmailBodyPartKnownFields struct {
	//PartId identifies this part uniquely within the Email. This is scoped to the emailId and has no meaning outside of the JMAP Email object representation. This is null if, and only if, the part is of type multipart/*.
	PartId *string `json:"partId"`

	//BlobId representing the raw octets of the contents of the part, after decoding any known Content-Transfer-Encoding (as defined in [@!RFC2045]), or null if, and only if, the part is of type multipart/*. Note that two parts may be transfer-encoded differently but have the same blob id if their decoded octets are identical and the server is using a secure hash of the data for the blob id. If the transfer encoding is unknown, it is treated as though it had no transfer encoding.
	BlobId *basetypes.Id `json:"blobId"`

	//Size, in octets, of the raw data after content transfer decoding (as referenced by the blobId, i.e., the number of octets in the file the user would download)
	Size basetypes.Uint `json:"size"`

	//This is a list of all header fields in the part, in the order they appear in the message. The values are in Raw form.
	Headers []EmailHeader `json:"headers"`

	//This is the decoded filename parameter of the Content-Disposition header field per [@!RFC2231], or (for compatibility with existing systems) if not present, then it’s the decoded name parameter of the Content-Type header field per [@!RFC2047].
	Name *string `json:"name"`

	//The value of the Content-Type header field of the part, if present; otherwise, the implicit type as per the MIME standard (text/plain or message/rfc822 if inside a multipart/digest). CFWS is removed and any parameters are stripped.
	Type *ContentType `json:"type"`

	//The value of the charset parameter of the Content-Type header field, if present, or null if the header field is present but not of type text/*. If there is no Content-Type header field, or it exists and is of type text/* but has no charset parameter, this is the implicit charset as per the MIME standard: us-ascii.
	CharSet *string `json:"charSet"`

	//The value of the charset parameter of the Content-Type header field, if present, or null if the header field is present but not of type text/*. If there is no Content-Type header field, or it exists and is of type text/* but has no charset parameter, this is the implicit charset as per the MIME standard: us-ascii.
	Disposition *string `json:"disposition"`

	//The value of the Content-Id header field of the part, if present; otherwise it’s null. CFWS and surrounding angle brackets (<>) are removed. This may be used to reference the content from within a text/html body part HTML using the cid: protocol, as defined in [@!RFC2392].
	Cid *string `json:"cid"`

	//The list of language tags, as defined in [@!RFC3282], in the Content-Language header field of the part, if present.
	Language []string `json:"language"`

	//The URI, as defined in [@!RFC2557], in the Content-Location header field of the part, if present.
	Location *string `json:"location"`

	//If the type is multipart/*, this contains the body parts of each child.
	SubParts []EmailBodyPart `json:"subParts"`
}

// In addition, the client may request/send EmailBodyPart properties representing individual header fields, following the same syntax and semantics as for the Email object, e.g., header:Content-Type.
type BespokeProperties map[string]any

type EmailBodyPart struct {
	EmailBodyPartKnownFields
	BespokeProperties
	//properties are the properties that need to be returned when marshalling
	properties []string
}

func (ebp EmailBodyPart) MarshalJSON() ([]byte, error) {
	//we need to do some merging of the known fields together with the fields in BespokeProperties
	//there must be a simpeler method than this
	edpBytes, err := json.Marshal(ebp.EmailBodyPartKnownFields)
	if err != nil {
		return nil, err
	}

	var edpMapStringAny = make(map[string]any, 0)

	if err := json.Unmarshal(edpBytes, &edpMapStringAny); err != nil {
		return nil, err
	}

	//remove all the fields we do not need except for subparts
	for k := range edpMapStringAny {
		if k == "subParts" {
			//although not made very explicit in the standard, we should always keep subParts
			continue
		}

		var keepProperty bool
		for _, p := range ebp.properties {
			if k == p {
				keepProperty = true
				break
			}
		}
		if !keepProperty {
			delete(edpMapStringAny, k)
		}
	}

	for k, v := range ebp.BespokeProperties {
		edpMapStringAny[k] = v
	}
	return json.Marshal(edpMapStringAny)
}

func (ja *JAccount) QueryEmail(ctx context.Context, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit int, calculateTotal bool, collapseThreads bool) (queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {

	ja.mlog.Debug("JAccount QueryEmail", slog.Any("collapseThreads", collapseThreads))

	q := bstore.QueryDB[store.Message](ctx, ja.mAccount.DB)

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

		default:
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("unsupported filter")
		}
	}

	for _, s := range sort {
		//FIXME we only support sorting at max one level
		switch s.Property {
		case "receivedAt":
			if s.IsAscending {
				q.SortAsc("Received")
			}
			q.SortDesc("Received")
		default:
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedSort("unsupported sort parameter")
		}
	}

	q.Limit(limit + int(position))

	q.FilterEqual("Deleted", false)
	q.FilterEqual("Expunged", false)

	var (
		//FIXME position can also be negative. In that case results need to come from the other end of the list.
		currentPos int64
		threadMap  map[int64]interface{} = make(map[int64]interface{})
	)

search:
	for {
		if !collapseThreads {
			var id int64
			if err := q.NextID(&id); err == bstore.ErrAbsent {
				// No more messages.
				// Note: if we don't iterate until an error, Close must be called on the query for cleanup.
				break search
			} else if err != nil {
				ja.mlog.Error("error getting next id", slog.Any("err", err.Error()))
				return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorServerFail()
			}

			// The ID is fetched from the index. The full record is
			// never read from the database. Calling Next instead
			// of NextID does always fetch, parse and return the
			// full record.
			if currentPos < int64(position) {
				continue search
			}
			currentPos++

			if len(ids) < limit {
				ids = append(ids, basetypes.NewIdFromInt64(id))
			}
			total++
		} else {
			msg, err := q.Next()
			if err == bstore.ErrAbsent {
				break search
			} else if err != nil {
				ja.mlog.Error("error getting message", slog.Any("err", err.Error()))
				return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorServerFail()
			}

			if _, ok := threadMap[msg.ThreadID]; !ok {

				if currentPos < int64(position) {
					continue search
				}

				if len(ids) < limit {
					ids = append(ids, basetypes.NewIdFromInt64(msg.ID))
				}
				threadMap[msg.ThreadID] = nil
				total++
			}
			currentPos++
		}
	}

	return "stubstate", false, position, ids, total, nil
}

func (ja *JAccount) GetEmail(ctx context.Context, ids []basetypes.Id, properties []string, bodyProperties []string, FetchTextBodyValues, FetchHTMLBodyValues, FetchAllBodyValues bool, MaxBodyValueBytes *basetypes.Uint) (state string, result []Email, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	ja.mlog.Debug("custom get params", slog.Any("bodyProperties", strings.Join(bodyProperties, ",")), slog.Any("FetchTextBodyValues", FetchTextBodyValues), slog.Any("FetchHTMLBodyValues", FetchHTMLBodyValues), slog.Any("FetchAllBodyValues", FetchAllBodyValues), slog.Any("MaxBodyValueBytes", MaxBodyValueBytes))

	for _, id := range ids {
		idInt64, err := id.Int64()
		if err != nil {
			//the email ids are imap ids meaning they are int64. When they cannot be converted to int64
			//we know that we never going to be able to return them
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
			ja.mlog.Error("error getting message from db", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		jem, merr := ja.NewEmail(em)
		if merr != nil {
			ja.mlog.Error("error instantiating new JEmail", slog.Any("id", idInt64), slog.Any("error", merr.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		if len(properties) == 0 {
			//no property list found so we return the default set as defined by the standard
			properties = defaultEmailPropertyFields
		}

		resultElement := Email{
			EmailDefinedProperties: EmailDefinedProperties{
				EmailMetadata: EmailMetadata{
					Id:         jem.Id(),
					ThreadId:   jem.ThreadId(),
					MailboxIds: jem.MailboxIds(),
					Size:       jem.Size(),
					ReceivedAt: jem.ReceivedAt(),
					Keywords:   jem.Keywords(),
					BlobId:     jem.Id(), //FIXME needs review
				},
				EmailBodyParts: EmailBodyParts{
					Preview: "<preview not available>",
				},
			},
			properties: properties,
		}

		var mErr *mlevelerrors.MethodLevelError

		resultElement.MessageId, mErr = jem.MessagedId()
		if mErr != nil {
			ja.mlog.Error("error getting messageId", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.SentAt, mErr = jem.SendAt()
		if mErr != nil {
			ja.mlog.Error("error getting date", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.Subject, mErr = jem.Subject()
		if mErr != nil {
			ja.mlog.Error("error getting subject", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.From, mErr = jem.From()
		if mErr != nil {
			ja.mlog.Error("error getting from", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.To, mErr = jem.To()
		if mErr != nil {
			ja.mlog.Error("error getting to", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.CC, mErr = jem.CC()
		if mErr != nil {
			ja.mlog.Error("error getting cc", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.BCC, mErr = jem.BCC()
		if mErr != nil {
			ja.mlog.Error("error getting bcc", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.BCC, mErr = jem.Sender()
		if mErr != nil {
			ja.mlog.Error("error getting sender", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.ReplyTo, mErr = jem.ReplyTo()
		if mErr != nil {
			ja.mlog.Error("error getting replyTo", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.InReplyTo, mErr = jem.InReplyTo()
		if mErr != nil {
			ja.mlog.Error("error getting inReplyTo", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.Preview, mErr = jem.Preview()
		if mErr != nil {
			ja.mlog.Error("error getting preview", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		resultElement.References, mErr = jem.References()
		if mErr != nil {
			ja.mlog.Error("error getting references", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}

		for _, prop := range properties {
			if strings.HasPrefix(prop, "header:") {
				//some custom headers are requested
				hParts := strings.Split(prop, ":")

				var headerName string
				var headerFormat string = "raw"
				var returnAll bool

				//if there are only 2 parts, then we use the fallback format which is raw
				if len(hParts) >= 2 {
					headerName = hParts[1]
				}
				if len(hParts) == 3 {
					headerFormat = hParts[2]
				}
				if len(hParts) == 4 {
					if hParts[3] == "all" {
						returnAll = true

					} else {
						continue
					}
				}
				if len(hParts) > 4 {
					//this format we do not recognize to skip it
					continue
				}

				if resultElement.DynamicProperties == nil {
					resultElement.DynamicProperties = make(map[string]any, 1)
				}

				resultElement.DynamicProperties[prop], mErr = jem.HeaderAs(ja.mlog, headerName, headerFormat, returnAll)
				if mErr != nil {
					ja.mlog.Error("error getting bespoke header", slog.Any("id", idInt64), slog.Any("prop", prop), slog.Any("error", err.Error()))
					return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
				}
			}
		}

		if HasAny(properties, "bodyStructure") {
			//FIXME In addition, the client may request/send EmailBodyPart properties representing individual header fields, following the same syntax and semantics as for the Email object, e.g., header:Content-Type.
			bs, mErr := jem.BodyStructure(bodyProperties)
			if mErr != nil {
				ja.mlog.Error("error getting body structure", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()

			}
			resultElement.BodyStructure = bs
		}

		if HasAny(properties, "bodyValues") {
			bvs, mErr := jem.BodyValues(FetchTextBodyValues, FetchHTMLBodyValues, FetchAllBodyValues, MaxBodyValueBytes)
			if mErr != nil {
				ja.mlog.Error("error getting body values", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			resultElement.BodyValues = bvs
		}

		if HasAny(properties, "textBody") {
			textBody, mErr := jem.HTMLBody(bodyProperties)
			if mErr != nil {
				ja.mlog.Error("error getting textBody", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			resultElement.TextBody = textBody
		}

		if HasAny(properties, "htmlBody") {
			htmlBody, mErr := jem.HTMLBody(bodyProperties)
			if mErr != nil {
				ja.mlog.Error("error getting htmlBody", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			resultElement.HTMLBody = htmlBody
		}

		if HasAny(properties, "attachments") {
			attachments, mErr := jem.Attachments(bodyProperties)
			if mErr != nil {
				ja.mlog.Error("error getting attachments", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			resultElement.Attachments = attachments
		}

		if HasAny(properties, "hasAttachment") {
			hasAttachment, mErr := jem.HasAttachment()
			if mErr != nil {
				ja.mlog.Error("error getting hasAttachment", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			resultElement.HasAttachment = hasAttachment
		}

		if HasAny(properties, "headers") {
			hdrs, mErr := jem.Headers()
			if mErr != nil {
				ja.mlog.Error("error getting headers", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			resultElement.Headers = hdrs
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

// HasAny returns true haystack has any needles
func HasAnyCaseInsensitive(haystack []string, needle ...string) bool {
	for _, h := range haystack {
		for _, n := range needle {
			if strings.ToLower(h) == strings.ToLower(n) {
				return true
			}
		}
	}
	return false
}

// JEmail is a helper object to efficiently return all the properties of the JMAP Email object to prevent a very long fn that does everything and is hard to test
type JEmail struct {
	//em is how the message is stored in db
	em store.Message

	//part is contains parsed parts of the message
	part message.Part

	logger mlog.Log

	//partsHaveBeenWalked is set to true when the subparts of part have been 'walked' meaning that the subparts have been populated
	partsHaveBeenWalked bool

	errorWhileWalkingParts bool
}

func NewJEmail(em store.Message, part message.Part, logger mlog.Log) JEmail {
	return JEmail{
		em:     em,
		part:   part,
		logger: logger,
	}
}

func (jem JEmail) Id() basetypes.Id {
	return basetypes.NewIdFromInt64(jem.em.ID)
}

func (jem JEmail) ThreadId() basetypes.Id {
	return basetypes.NewIdFromInt64(jem.em.ThreadID)
}

func (jem JEmail) MailboxIds() map[basetypes.Id]bool {
	return map[basetypes.Id]bool{
		basetypes.NewIdFromInt64(jem.em.MailboxID): true,
	}
}

func (jem JEmail) Size() basetypes.Uint {
	return basetypes.Uint(jem.em.Size)
}

func (jem JEmail) ReceivedAt() basetypes.UTCDate {
	return basetypes.UTCDate(jem.em.Received)
}

func (jem JEmail) Keywords() map[string]bool {
	f := jem.em.Flags

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

// MessagedId returns the messageId property
func (jem JEmail) MessagedId() ([]string, *mlevelerrors.MethodLevelError) {
	msgIDsIface, merr := jem.HeaderAs(jem.logger, "Message-ID", "asMessageIds", false)

	if msgIDs, ok := msgIDsIface.([]string); ok {
		return msgIDs, nil
	}

	return nil, merr
}

// InReplyTo returns inReplyTo
func (jem JEmail) InReplyTo() ([]string, *mlevelerrors.MethodLevelError) {
	msgIDsIface, merr := jem.HeaderAs(jem.logger, "In-Reply-To", "asMessageIds", false)
	if msgIDs, ok := msgIDsIface.([]string); ok {
		return msgIDs, nil
	}

	return nil, merr
}

// Date returns date
func (jem JEmail) SendAt() (*basetypes.Date, *mlevelerrors.MethodLevelError) {
	if env := jem.part.Envelope; env != nil {
		result := basetypes.Date(env.Date)
		return &result, nil
	}
	return nil, nil
}

// Subject returns the subject property
func (jem JEmail) Subject() (*string, *mlevelerrors.MethodLevelError) {
	if env := jem.part.Envelope; env != nil {
		return &env.Subject, nil
	}
	return nil, nil
}

// From returns from
func (jem JEmail) From() ([]EmailAddress, *mlevelerrors.MethodLevelError) {
	var result []EmailAddress

	if env := jem.part.Envelope; env != nil {
		for _, addr := range env.From {
			result = append(result, msgAddressToEmailAddress(addr))
		}
		return result, nil
	}
	return nil, nil
}

// To returns to
func (jem JEmail) To() ([]EmailAddress, *mlevelerrors.MethodLevelError) {
	var result []EmailAddress

	if env := jem.part.Envelope; env != nil {
		for _, addr := range env.To {
			result = append(result, msgAddressToEmailAddress(addr))
		}
		return result, nil
	}
	return nil, nil
}

// CC returns cc
func (jem JEmail) CC() ([]EmailAddress, *mlevelerrors.MethodLevelError) {
	var result []EmailAddress

	if env := jem.part.Envelope; env != nil {
		for _, addr := range env.CC {
			result = append(result, msgAddressToEmailAddress(addr))
		}
		return result, nil
	}
	return nil, nil
}

// BCC returns bcc
func (jem JEmail) BCC() ([]EmailAddress, *mlevelerrors.MethodLevelError) {
	var result []EmailAddress

	if env := jem.part.Envelope; env != nil {
		for _, addr := range env.BCC {
			result = append(result, msgAddressToEmailAddress(addr))
		}
		return result, nil
	}
	return nil, nil
}

// Sender returns sender
func (jem JEmail) Sender() ([]EmailAddress, *mlevelerrors.MethodLevelError) {

	var result []EmailAddress

	if env := jem.part.Envelope; env != nil {
		for _, addr := range env.Sender {
			result = append(result, msgAddressToEmailAddress(addr))
		}
		return result, nil
	}
	return nil, nil
}

// ReplyTo returns reply to addresses
func (jem JEmail) ReplyTo() ([]EmailAddress, *mlevelerrors.MethodLevelError) {

	var result []EmailAddress

	if env := jem.part.Envelope; env != nil {
		for _, addr := range env.ReplyTo {
			result = append(result, msgAddressToEmailAddress(addr))
		}
		return result, nil
	}
	return nil, nil
}

// References return the RFC822 header with the same name
func (jem JEmail) References() ([]string, *mlevelerrors.MethodLevelError) {

	result, merr := jem.HeaderAs(jem.logger, "References", "asMessageIds", false)
	if merr != nil {
		return nil, merr
	}
	if result == nil {
		return nil, nil
	}
	if resultStringSlice, ok := result.([]string); ok {
		return resultStringSlice, nil
	}
	return nil, nil

}

// HeaderAs returns a header in a specific format
func (jem JEmail) HeaderAs(mlog mlog.Log, headerName string, format string, retAll bool) (any, *mlevelerrors.MethodLevelError) {
	orderedHeaders, err := jem.part.HeaderInOrder()
	if err != nil {
		jem.logger.Error("getting ordered headers failed", slog.Any("err", err.Error()))
		return "", mlevelerrors.NewMethodLevelErrorServerFail()
	}

	//return nil if empty header
	if retAll {
		if orderedHeaders.Last(headerName) == "" {
			return nil, nil
		}
	}

	headerFieldsDefinedInRFC5322RFC2369 := []string{
		"orig-date",     //RFC 5322 3.6.1
		"from",          //RFC 5322 3.6.2
		"sender",        //RFC 5322 3.6.2
		"reply-to",      //RFC 5322 3.6.2
		"to",            //RFC 5322 3.6.3
		"cc",            //RFC 5322 3.6.3
		"bcc",           //RFC 5322 3.6.3
		"message-id",    //RFC 5322 3.6.4
		"in-reply-to",   //RFC 5322 3.6.4
		"references",    //RFC 5322 3.6.4
		"subject",       //RFC 5322 3.6.5
		"comments",      //RFC 5322 3.6.5
		"keywords",      //RFC 5322 3.6.5
		"resent-date",   //RFC 5322 3.6.6
		"resent-from",   //RFC 5322 3.6.6
		"resent-to",     //RFC 5322 3.6.6
		"resent-cc",     //RFC 5322 3.6.6
		"resent-bcc",    //RFC 5322 3.6.6
		"resent-msg-id", //RFC 5322 3.6.6
		"return",        //RFC 5322 3.6.7
		"received",      //RFC 5322 3.6.7

		"list-help",        //RFC 2369 3.x
		"list-unsubscribe", //RFC 2369 3.x
		"list-subscribe",   //RFC 2369 3.x
		"list-post",        //RFC 2369 3.x
		"list-owner",       //RFC 2369 3.x
		"list-archive",     //RFC 2369 3.x

	}

	switch format {
	case "asRaw":
		//The raw octets of the header field value from the first octet following the header field name terminating colon, up to but excluding the header field terminating CRLF. Any standards-compliant message MUST be either ASCII (RFC 5322) or UTF-8 (RFC 6532); however, other encodings exist in the wild. A server SHOULD replace any octet or octet run with the high bit set that violates UTF-8 syntax with the unicode replacement character (U+FFFD). Any NUL octet MUST be dropped.
		//FIXME this header is already parsed . I need to find a solution for this
		if retAll {
			return orderedHeaders.Values(headerName), nil
		}
		return orderedHeaders.Last(headerName), nil
	case "asText":
		if HasAnyCaseInsensitive([]string{"subject", "comments", "keywords", "list-id"}, headerName) || !HasAnyCaseInsensitive(headerFieldsDefinedInRFC5322RFC2369, headerName) {
			if retAll {
				return orderedHeaders.Values(headerName), nil
			}
			return orderedHeaders.Last(headerName), nil
		}
	case "asAddresses":
		if HasAnyCaseInsensitive([]string{"from", "sender", "reply-to", "to", "cc", "bcc", "resent-from", "resent-sender", "resent-reply-to", "resent-to", "resent-cc", "resent-bcc"}, headerName) || !HasAnyCaseInsensitive(headerFieldsDefinedInRFC5322RFC2369, headerName) {
			var result []EmailAddress

			if !retAll {
				for _, addr := range message.ParseAddressList(jem.logger, mail.Header(orderedHeaders.MIMEHeader()), headerName) {
					result = append(result, msgAddressToEmailAddress(addr))
				}
			} else {
				//FIXME cannot reuse ParseAddressList here
			}
			return result, nil
		}
	case "asGroupedAddresses":
		//same condidtions as asAddresses
		if HasAnyCaseInsensitive([]string{"from", "sender", "reply-to", "to", "cc", "bcc", "resent-from", "resent-sender", "resent-reply-to", "resent-to", "resent-cc", " resent-bcc"}, headerName) || !HasAnyCaseInsensitive(headerFieldsDefinedInRFC5322RFC2369, headerName) {
			//FIXME this is not supported (yet?) in mox
		}
	case "asMessageIds":
		//The header field is parsed as a list of msg-id values, as specified in [@!RFC5322], Section 3.6.4, into the String[] type. Comments and/or folding white space (CFWS) and surrounding angle brackets (<>) are removed. If parsing fails, the value is null.
		if HasAnyCaseInsensitive([]string{"message-id", "in-reply-to", "references", "resent-message-id"}, headerName) || !HasAnyCaseInsensitive(headerFieldsDefinedInRFC5322RFC2369, headerName) {
			submatches := regexp.MustCompile("<(\\S+)>").FindStringSubmatch(orderedHeaders.Last(headerName))

			if len(submatches) == 2 {
				return submatches[1:], nil
			}

			//FIXME: need to implement retAll
		}
	case "asDate":
		if HasAnyCaseInsensitive([]string{"date", "resent-date"}, headerName) || !HasAnyCaseInsensitive(headerFieldsDefinedInRFC5322RFC2369, headerName) {
			if val := orderedHeaders.Last(headerName); val != "" {
				d, err := mail.ParseDate(val)
				if err == nil {
					return basetypes.Date(d), nil
				}
			}
			//FIXME: need to implement retAll
		}
	case "asURLs":
		if HasAnyCaseInsensitive([]string{"list-help", "list-unsubscribe", "list-post", "list-owner", "list-archive"}, headerName) || !HasAnyCaseInsensitive(headerFieldsDefinedInRFC5322RFC2369, headerName) {
			var result []string
			for _, headerVal := range orderedHeaders.Values(headerName) {
				if headerVal != "" {
					result = append(result, regexp.MustCompile("<(\\S+>)").FindAllString(headerVal, -1)...)
				}
				return result, nil
			}
			//FIXME: need to implement retAll
		}
	default:
		return nil, nil
	}
	return nil, nil
}

func (jem JEmail) Preview() (string, *mlevelerrors.MethodLevelError) {
	//FIXME this needs to be more specific and take into account that we can have multipart also in the first subpart

	partForPreview := jem.part
	if len(jem.part.Parts) > 0 {
		partForPreview = jem.part.Parts[0]
	}

	//read the whole body and see what we got
	fullBody, err := io.ReadAll(partForPreview.Reader())
	if err != nil {
		return "", mlevelerrors.NewMethodLevelErrorServerFail()
	}
	if len(fullBody) < 100 {
		return string(fullBody), nil
	}
	return string(fullBody[:100]), nil
}

func (jem JEmail) BodyStructure(bodyProperties []string) (EmailBodyPart, *mlevelerrors.MethodLevelError) {

	partID := 0

	//do the top level part first
	result := partToEmailBodyPart(jem.part, &partID, jem.em.ID, bodyProperties)

	//recurse over the subparts
	recursePartToEmailBodyPart(jem.part.Parts, jem.em.ID, bodyProperties, &result, &partID)

	return result, nil
}

// partToEmailBodyPart returns the EmailBodyPart for the part (type message.Part)
func partToEmailBodyPart(part message.Part, nextPartID *int, idInt64 int64, bodyProperties []string) EmailBodyPart {
	ebd := EmailBodyPart{
		EmailBodyPartKnownFields: EmailBodyPartKnownFields{},
		properties:               bodyProperties,
	}

	jPart, headerParseErr := newJPartLegacy(part, idInt64, *nextPartID)
	ebd.Size = jPart.Size() //TODO: are these properties here on purpose?
	ebd.Cid = jPart.Cid()

	if headerParseErr == nil {
		ebd.Headers = jPart.Headers()
		ebd.Disposition = jPart.Disposition()
		ebd.Name = jPart.Name()
		ebd.Type = jPart.Type()
		ebd.CharSet = jPart.Charset()
		ebd.Location = jPart.Location()
		ebd.Language = jPart.Language()

		if ebd.Type != nil && !ebd.Type.IsMultipart() {
			//This is null if, and only if, the part is of type multipart/*
			partIDStr := fmt.Sprintf("%d", *nextPartID)
			ebd.PartId = &partIDStr

			//increase the partID counter

			//BlobId
			//FIXME just choosing a way to store things
			//we have to come up with a way how to generate this

			blobId := basetypes.Id(fmt.Sprintf("%d-%s", idInt64, partIDStr))
			ebd.BlobId = &blobId

			*nextPartID++
		}
	}
	return ebd
}

func recursePartToEmailBodyPart(subparts []message.Part, idInt64 int64, bodyProperties []string, result *EmailBodyPart, nextPartId *int) {
	if len(subparts) == 0 {
		return
	} else {

		for _, p := range subparts {
			subPartBodyPart := partToEmailBodyPart(p, nextPartId, idInt64, bodyProperties)

			if subPartBodyPart.Type != nil {
				//This is the full MIME structure of the message body, without recursing into message/rfc822 or message/global parts
				if *subPartBodyPart.Type == "message/rfc822" || *subPartBodyPart.Type == "message/global" {
					continue
				}
			}

			recursePartToEmailBodyPart(p.Parts, idInt64, bodyProperties, &subPartBodyPart, nextPartId)

			result.SubParts = append(result.SubParts, subPartBodyPart)
		}
	}
}

func (jem JEmail) GetPartBody(partIDToLookFor string) (string, *mlevelerrors.MethodLevelError) {
	//this can later be reused to get a particular BlobId
	//since BlobIds have a Global Scope, we need to add a prefix

	//FIXME I would need the structure so I can parse at least the content type

	nextNum := 0
	return searchPartRecursive(partIDToLookFor, jem.part, &nextNum)

}

func searchPartRecursive(partID string, part message.Part, nextNum *int) (string, *mlevelerrors.MethodLevelError) {
	//FIXME need an error to indicate the part was not found
	if part.MediaType != "MULTIPART" {
		if partID == fmt.Sprintf("%d", *nextNum) {
			fullBody, err := io.ReadAll(part.Reader())
			if err != nil {
				return "", mlevelerrors.NewMethodLevelErrorServerFail()
			}
			return string(fullBody), nil
		}
		*nextNum++
	}
	for _, subPart := range part.Parts {
		body, err := searchPartRecursive(partID, subPart, nextNum)
		if err == nil && body != "" {
			return body, err
		}
	}
	return "", nil
}

func (jem JEmail) BodyValues(fetchTextBodyValues, fetchHTMLBodyValues, fetchAllBodyValues bool, maxBodyValueBytes *basetypes.Uint) (map[string]EmailBodyValue, *mlevelerrors.MethodLevelError) {
	//This is a map of partId to an EmailBodyValue object for none, some, or all text/* parts. Which parts are included and whether the value is truncated is determined by various arguments to Email/get and Email/parse.

	result := make(map[string]EmailBodyValue, 0)

	uniquePartsToGet := make(map[string]any, 0)

	//fetchAllBodyValues is a combination of fetchTextBodyValues and fetchHTMLBodyValues

	//the approach is that we first determine which parts to get and then to get them
	if fetchTextBodyValues || fetchAllBodyValues {
		//get the part ids
		textBodyParts, mErr := jem.TextBody([]string{"partId"})
		if mErr != nil {
			return nil, mErr
		}

		for _, bp := range textBodyParts {
			if bp.Type != nil && bp.Type.IsText() {
				uniquePartsToGet[*bp.PartId] = nil
			}
		}
	}

	if fetchHTMLBodyValues || fetchAllBodyValues {
		htmlBodyParts, mErr := jem.HTMLBody([]string{"partId"})
		if mErr != nil {
			return nil, mErr
		}

		for _, bp := range htmlBodyParts {
			if bp.Type != nil && bp.Type.IsText() {
				uniquePartsToGet[*bp.PartId] = nil
			}
		}
	}

	for partId := range uniquePartsToGet {
		bodyVal, mErr := jem.GetPartBody(partId)
		if mErr != nil {
			return nil, mErr
		}

		//FIXME make sure not to cut in a HREF link
		var truncated bool
		if maxBodyValueBytes != nil {
			if len(bodyVal) > int(*maxBodyValueBytes) {
				bodyVal = string(bodyVal[:*maxBodyValueBytes])
				truncated = true
			}
		}

		result[partId] = EmailBodyValue{
			Value:       bodyVal,
			IsTruncated: truncated,
		}
	}
	return result, nil
}

// TextBody returns a list of EmailBodyParts of type text/plain, text/html, image/*, audio/*, and/or video/* parts to display (sequentially) as the message body, with a preference for text/plain when alternative versions are available.
func (jem JEmail) TextBody(bodyProperties []string) ([]EmailBodyPart, *mlevelerrors.MethodLevelError) {
	// A list of text/plain, text/html, image/*, audio/*, and/or video/* parts to display (sequentially) as the message body, with a preference for text/plain when alternative versions are available.

	nextPartID := 0
	return flattenPartToEmailBodyPart(jem.part, jem.em.ID, &nextPartID, bodyProperties, flattenTypeText), nil
}

// TextBody returns a list of EmailBodyParts of type text/plain, text/html, image/*, audio/*, and/or video/* parts to display (sequentially) as the message body, with a preference for text/html when alternative versions are available.
func (jem JEmail) HTMLBody(bodyProperties []string) ([]EmailBodyPart, *mlevelerrors.MethodLevelError) {
	//A list of text/plain, text/html, image/*, audio/*, and/or video/* parts to display (sequentially) as the message body, with a preference for text/html when alternative versions are available.
	nextPartID := 0
	return flattenPartToEmailBodyPart(jem.part, jem.em.ID, &nextPartID, bodyProperties, flattenTypeHTML), nil
}

func (jem JEmail) Attachments(bodyProperties []string) ([]EmailBodyPart, *mlevelerrors.MethodLevelError) {
	return jem.attachments(bodyProperties)
}

func (jem JEmail) attachments(bodyProperties []string) ([]EmailBodyPart, *mlevelerrors.MethodLevelError) {
	/*
		A list, traversing depth-first, of all parts in bodyStructure that satisfy either of the following conditions:
		- not of type multipart/* and not included in textBody or htmlBody
		- of type image/*, audio/*, or video/* and not in both textBody and htmlBody


		FIXME: it should be a lot simpeler defining what the above output is other than implementing the lines literally like below
	*/

	var result []EmailBodyPart

	textBodyParts, mErr := jem.TextBody(bodyProperties)
	if mErr != nil {
		return result, mErr
	}

	htmlBodyParts, mErr := jem.HTMLBody(bodyProperties)
	if mErr != nil {
		return result, mErr
	}

	nextPartID := 0
	nonMultiPartParts, mErr := getPartsNotOfTypeMultipart(jem.part, jem.em.ID, &nextPartID, bodyProperties), nil
	if mErr != nil {
		return result, mErr
	}

loopNonMultipartResultCandidates:
	for _, resultCandidate := range nonMultiPartParts {
		for _, textBodyPart := range textBodyParts {
			if *textBodyPart.PartId == *resultCandidate.PartId {
				continue loopNonMultipartResultCandidates
			}
		}
		for _, htmlBodyPart := range htmlBodyParts {
			if *htmlBodyPart.PartId == *resultCandidate.PartId {
				continue loopNonMultipartResultCandidates
			}
		}

		result = append(result, resultCandidate)
	}

	nextPartID = 0
	vidAudImgParts := getPartsOfTypeImageAudioVideo(jem.part, jem.em.ID, &nextPartID, bodyProperties)

loopMediaFiles:
	for _, resultCandidate := range vidAudImgParts {
		var inTextBodyParts, inHTMLBodyParts bool

		for _, textBodyPart := range textBodyParts {
			if *textBodyPart.PartId == *resultCandidate.PartId {
				inTextBodyParts = true
			}
		}
		for _, htmlBodyPart := range htmlBodyParts {
			if *htmlBodyPart.PartId == *resultCandidate.PartId {
				inHTMLBodyParts = true
			}
		}

		if !inTextBodyParts && !inHTMLBodyParts {
			//we can potentially include it unless we already have it
			for _, resultItem := range result {
				if *resultItem.PartId == *resultCandidate.PartId {
					continue loopMediaFiles
				}
			}
			result = append(result, resultCandidate)
		}

	}
	//FIXME need to sort the result on partID

	return result, nil
}

// parseBodyParts is the implementation of the reference at https://jmap.io/spec-mail.html
// AAA
func (jem JEmail) parseBodyParts(properties []string) (textBody, htmlBody, attachments []EmailBodyPart, mErr *mlevelerrors.MethodLevelError) {
	jPart, jPartErr := jem.JPart()
	if jPartErr != nil {
		return nil, nil, nil, jPartErr
	}

	var textBodyParts, htmlBodyParts, attachmentParts []JPart

	if mErr := parseBodyPartsRecursive([]JPart{*jPart}, "mixed", false, textBodyParts, htmlBodyParts, attachmentParts); mErr != nil {
		return nil, nil, nil, mErr
	}

	for _, tPart := range textBodyParts {
		textBody = append(textBody, tPart.EmailBodyPart(properties))
	}
	for _, hPart := range htmlBodyParts {
		htmlBody = append(htmlBody, hPart.EmailBodyPart(properties))
	}
	for _, att := range attachmentParts {
		attachments = append(attachments, att.EmailBodyPart(properties))
	}

	return textBody, htmlBody, attachments, nil
}

// parseBodyPartsRecursive is the implementation of the reference at https://jmap.io/spec-mail.html
func parseBodyPartsRecursive(parts []JPart, multipartType string, inAlternative bool, textBody, htmlBody, attachments []JPart) *mlevelerrors.MethodLevelError {

	lenghtOrMinus1 := func(bp []JPart) int {
		if lenBP := len(bp); lenBP == 0 {
			return -1
		} else {
			return lenBP
		}
	}

	isInlineMediaPart := func(part JPart) bool {
		return part.Type().IsImage() ||
			part.Type().IsAudio() ||
			part.Type().IsVideo()
	}

	textLength := lenghtOrMinus1(textBody)
	htmlLength := lenghtOrMinus1(htmlBody)

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		disposition := part.Disposition()
		isInLine := (disposition != nil && *disposition == "attachment") &&
			(part.Type().IsTextPlain() ||
				part.Type().IsTextHTML() ||
				isInlineMediaPart(part)) && (i == 0 || (!part.Type().IsMultpartRelated() && (isInlineMediaPart(part) || part.Name() == nil)))

		if part.Type().IsMultipart() {
			_, subMultiType, ok := strings.Cut(part.Type().String(), "/")
			if !ok {
				return mlevelerrors.NewMethodLevelErrorServerFail()
			}
			return parseBodyPartsRecursive(part.JParts, subMultiType, inAlternative || subMultiType == "alternative", htmlBody, textBody, attachments)

		} else if isInLine {
			if multipartType == "alternative" {
				switch {
				case part.Type().IsTextPlain():
					//push to textbody
					textBody = append(textBody, part)
				case part.Type().IsTextHTML():
					//push to textbody
					htmlBody = append(htmlBody, part)
				default:
					//push to attachments
					attachments = append(attachments, part)
				}
				continue
			} else if inAlternative {
				if part.Type().IsTextPlain() {
					htmlBody = nil
				}
				if part.Type().IsTextHTML() {
					textBody = nil
				}
			}

			if textBody != nil {
				textBody = append(textBody, part)
			}

			if htmlBody != nil {
				htmlBody = append(htmlBody, part)
			}

			if (textBody == nil || htmlBody == nil) && isInlineMediaPart(part) {
				attachments = append(attachments, part)
			}
		} else {
			attachments = append(attachments, part)
		}

	}

	if multipartType == "alternative" && textBody != nil && htmlBody != nil {
		if textLength == len(textBody) && htmlLength != len(htmlBody) {
			for i := htmlLength; i < len(htmlBody); i++ {
				textBody = append(textBody, htmlBody[i])
			}
		}
		if htmlLength == len(htmlBody) && textLength != len(textBody) {
			for i := textLength; i < len(textBody); i++ {
				htmlBody = append(htmlBody, textBody[i])
			}

		}
	}
	return nil
}

func (jem JEmail) HasAttachment() (bool, *mlevelerrors.MethodLevelError) {
	/*
		This is true if there are one or more parts in the message that a client UI should offer as downloadable. A server SHOULD set hasAttachment to true if the attachments list contains at least one item that does not have Content-Disposition: inline. The server MAY ignore parts in this list that are processed automatically in some way or are referenced as embedded images in one of the text/html parts of the message.
	*/

	attachments, mErr := jem.attachments([]string{"disposition"})
	if mErr != nil {
		return false, mErr
	}

	for _, attachment := range attachments {
		if attachment.Disposition != nil && *attachment.Disposition != "inline" {
			return true, nil
		}
	}
	return false, nil
}

func (jem JEmail) Headers() ([]EmailHeader, *mlevelerrors.MethodLevelError) {

	headers, err := jem.part.HeaderInOrder()
	if err != nil {
		jem.logger.Logger.Error("error getting headers of email", slog.Any("id", jem.Id()), slog.Any("err", err))
		return nil, mlevelerrors.NewMethodLevelErrorServerFail()
	}
	var result []EmailHeader
	for _, hdr := range headers {
		result = append(result, EmailHeader{
			Name:  hdr.Name,
			Value: hdr.Value,
		})
	}
	return result, nil

}

func getPartsNotOfTypeMultipart(part message.Part, messageIDInt64 int64, nextPartID *int, bodyProperties []string) []EmailBodyPart {
	var result []EmailBodyPart

	topLevelPart := partToEmailBodyPart(part, nextPartID, messageIDInt64, bodyProperties)
	if topLevelPart.Type != nil {
		if !topLevelPart.Type.IsMultipart() {
			result = append(result, partToEmailBodyPart(part, nextPartID, messageIDInt64, bodyProperties))
		}
	}

	for _, p := range part.Parts {
		result = append(result, getPartsNotOfTypeMultipart(p, messageIDInt64, nextPartID, bodyProperties)...)
	}

	return result
}

func getPartsOfTypeImageAudioVideo(part message.Part, messageIDInt64 int64, nextPartID *int, bodyProperties []string) []EmailBodyPart {
	var result []EmailBodyPart

	topLevelPart := partToEmailBodyPart(part, nextPartID, messageIDInt64, bodyProperties)
	if topLevelPart.Type != nil {
		if topLevelPart.Type.IsVideo() || topLevelPart.Type.IsAudio() || topLevelPart.Type.IsImage() {
			result = append(result, partToEmailBodyPart(part, nextPartID, messageIDInt64, bodyProperties))
		} else if !topLevelPart.Type.IsMultipart() {
			//we must keep the partIDs correct
			*nextPartID++
		}
	}

	for _, p := range part.Parts {
		result = append(result, getPartsOfTypeImageAudioVideo(p, messageIDInt64, nextPartID, bodyProperties)...)
	}

	return result
}

// includeFunc is called in flattenPartToEmailBodyPart to instruct to include/exclude a particular part from in the result
type flattenType int

const (
	flattenTypeText flattenType = iota
	flattenTypeHTML
)

func flattenPartToEmailBodyPart(part message.Part, messageIDInt64 int64, nextPartID *int, bodyProperties []string, flattenType flattenType) []EmailBodyPart {
	//FIXME need to recurse and support flattenTypeAttachments

	var result []EmailBodyPart

	topLevelPart := partToEmailBodyPart(part, nextPartID, messageIDInt64, bodyProperties)

	//FIXME this is far from complete but we need something as a start

	if topLevelPart.Type != nil {

		switch {
		case topLevelPart.Type.IsMultpartAlternative():
			for _, p := range part.Parts {
				partBodyPart := partToEmailBodyPart(p, nextPartID, messageIDInt64, bodyProperties)

				if partBodyPart.Type != nil {
					switch flattenType {
					case flattenTypeHTML:
						if partBodyPart.Type.IsTextHTML() {
							result = append(result, partBodyPart)
						}
					case flattenTypeText:
						if partBodyPart.Type.IsTextPlain() {
							result = append(result, partBodyPart)
						}
					}

				}
				for _, pParts := range p.Parts {
					result = append(result, flattenPartToEmailBodyPart(pParts, messageIDInt64, nextPartID, bodyProperties, flattenType)...)
				}
			}
		default:
			var include bool
			switch {
			case topLevelPart.Type.IsTextHTML() || topLevelPart.Type.IsTextPlain():
				include = true
			case topLevelPart.Type.IsImage(), topLevelPart.Type.IsAudio(), topLevelPart.Type.IsVideo():
				include = true
			}

			if include {
				result = append(result, topLevelPart)
			}

			for _, p := range part.Parts {
				result = append(result, flattenPartToEmailBodyPart(p, messageIDInt64, nextPartID, bodyProperties, flattenType)...)
			}

		}

	}

	return result
}

// JPart is a helper to get the BodyPart properties we need
type JPart struct {
	p message.Part

	//id is the id of the part. The id identifies the part
	id string

	//msgID is used for the blobid
	msgID         int64
	headerInOrder message.HeaderInOrder
	JParts        []JPart
}

// JPart returns a JPart representation of a JEmail
func (jem JEmail) JPart() (*JPart, *mlevelerrors.MethodLevelError) {
	id := 0
	result, mErr := newJPart(jem.part, jem.em.ID, &id)
	if mErr != nil {
		return nil, mErr
	}

	if mErr := partToJPartRecurse(result, jem.em.ID, &id); mErr != nil {
		return nil, mErr
	}
	return result, nil
}

func partToJPartRecurse(p *JPart, messageID int64, id *int) *mlevelerrors.MethodLevelError {
	for _, part := range p.p.Parts {
		newJPart, mErr := newJPart(part, messageID, id)
		if mErr != nil {
			return mErr
		}

		if mErr := partToJPartRecurse(newJPart, messageID, id); mErr != nil {
			return mErr
		}
		p.JParts = append(p.JParts, *newJPart)
	}
	return nil
}

func newJPart(p message.Part, messageID int64, nextID *int) (*JPart, *mlevelerrors.MethodLevelError) {
	result := &JPart{
		p:     p,
		msgID: messageID,
	}

	headers, err := p.HeaderInOrder()
	if err != nil {
		return result, mlevelerrors.NewMethodLevelErrorServerFail()
	}
	result.headerInOrder = headers

	//this is weird: the constructor kind of depends on it self
	if t := result.Type(); t != nil && !t.IsMultipart() {
		result.id = fmt.Sprintf("%d", *nextID)
		//if used then increment
		*nextID += 1
	}

	return result, nil
}

func newJPartLegacy(p message.Part, messageID int64, id int) (*JPart, *mlevelerrors.MethodLevelError) {
	result := &JPart{
		p: p,
		//id:    id,
		msgID: messageID,
	}

	headers, err := p.HeaderInOrder()
	if err != nil {
		return result, mlevelerrors.NewMethodLevelErrorServerFail()
	}
	result.headerInOrder = headers

	return result, nil
}

func (jp JPart) ID() string {
	return jp.id
}

func (jp JPart) BlobID() basetypes.Id {
	return basetypes.Id(fmt.Sprintf("%d-%s", jp.msgID, jp.id))
}

func (jp JPart) Size() basetypes.Uint {
	return basetypes.Uint(jp.p.BodyOffset - jp.p.HeaderOffset)
}

func (jp JPart) Cid() *string {
	if jp.p.ContentID == "" {
		return nil
	}
	//brackets need to go
	result := strings.TrimPrefix(jp.p.ContentID, "<")
	result = strings.TrimSuffix(result, ">")
	return &result
}

func (jp JPart) Headers() []EmailHeader {
	var result []EmailHeader
	for _, h := range jp.headerInOrder {
		result = append(result, EmailHeader{
			Name:  h.Name,
			Value: h.Value,
		})
	}
	return result
}

func (jp JPart) Disposition() *string {
	if jp.headerInOrder != nil {
		val := jp.headerInOrder.Last("Content-Disposition")
		if val != "" {
			dispVal, _, err := mime.ParseMediaType(val)
			if err == nil {
				//disposition
				return &dispVal
			}
		}
	}
	return nil
}

func (jp JPart) Name() *string {
	if jp.headerInOrder != nil {
		val := jp.headerInOrder.Last("Content-Disposition")
		if val != "" {
			_, params, err := mime.ParseMediaType(val)
			if err == nil {
				//Name
				fileName, ok := params["filename"]
				if ok {
					return &fileName
				}

				//name fallback
				val := jp.headerInOrder.Last("Content-Type")
				if val != "" {
					_, params, err := mime.ParseMediaType(val)
					if err == nil {
						name, ok := params["name"]
						if ok {
							return &name
						}
					}
				}
			}
		}
	}
	return nil
}

type ContentType string

func NewContentType(t string) ContentType {
	return ContentType(t)
}

func (ct ContentType) IsMultipart() bool {
	return strings.HasPrefix(string(ct), "multipart/")
}

func (ct ContentType) IsText() bool {
	return strings.HasPrefix(string(ct), "text/")
}

func (ct ContentType) IsAudio() bool {
	return strings.HasPrefix(string(ct), "audio/")
}

func (ct ContentType) IsVideo() bool {
	return strings.HasPrefix(string(ct), "video/")
}

func (ct ContentType) IsImage() bool {
	return strings.HasPrefix(string(ct), "image/")
}

func (ct ContentType) IsMultpartAlternative() bool {
	return ct == "multipart/alternative"
}

func (ct ContentType) IsMultpartRelated() bool {
	return ct == "multipart/related"
}

func (ct ContentType) IsTextPlain() bool {
	return ct == "text/plain"
}

func (ct ContentType) IsTextHTML() bool {
	return ct == "text/html"
}

func (ct ContentType) String() string {
	return string(ct)
}

func (jp JPart) Type() *ContentType {
	if jp.headerInOrder != nil {
		if val := jp.headerInOrder.Last("Content-Type"); val != "" {
			mediaType, _, err := mime.ParseMediaType(val)
			if err == nil {
				ct := NewContentType(mediaType)
				return &ct
			}
		}
	}
	return nil
}

func (jp JPart) Charset() *string {
	if jp.headerInOrder != nil {
		if val := jp.headerInOrder.Last("Content-Type"); val != "" {
			mediaType, params, err := mime.ParseMediaType(val)
			if err == nil {
				//charset
				if strings.HasPrefix(mediaType, "text/") {
					if charset, ok := params["charset"]; ok {
						return &charset
					} else {
						fallbackCharSet := "us-ascii"
						return &fallbackCharSet
					}
				}
			}
		}
	}
	return nil
}

func (jp JPart) Location() *string {
	//FIXME need to validate this is correct
	if jp.headerInOrder != nil {
		if loc := jp.headerInOrder.Last("Content-Location"); loc != "" {
			return &loc
		}
	}
	return nil
}

func (jp JPart) Language() []string {
	//FIXME need to check if I need to remove comment kind of things here
	var result []string
	if jp.headerInOrder != nil {
		if languages := jp.headerInOrder.Last("Content-Language"); languages != "" {
			for _, l := range strings.Split(languages, ",") {
				result = append(result, strings.Trim(l, " "))
			}
		}
	}
	return result
}

func (jp JPart) EmailBodyPart(bodyProperties []string) EmailBodyPart {
	ebd := EmailBodyPart{
		EmailBodyPartKnownFields: EmailBodyPartKnownFields{
			Size:        jp.Size(),
			Headers:     jp.Headers(),
			Name:        jp.Name(),
			Type:        jp.Type(),
			CharSet:     jp.Charset(),
			Disposition: jp.Disposition(),
			Cid:         jp.Cid(),
			Language:    jp.Language(),
			Location:    jp.Location(),
			//FIXE this is trouble some if you ask me when returning textBody/htmlBody/attachments
			SubParts: []EmailBodyPart{},
		},
		properties: bodyProperties,
	}

	if id := jp.ID(); id != "" {
		ebd.PartId = &id

		blobID := jp.BlobID()
		ebd.BlobId = &blobID
	}
	return ebd
}

func (jp JPart) Body() (string, *mlevelerrors.MethodLevelError) {
	body, err := io.ReadAll(jp.p.Reader())
	if err != nil {
		return "", mlevelerrors.NewMethodLevelErrorServerFail()
	}
	return string(body), nil
}
