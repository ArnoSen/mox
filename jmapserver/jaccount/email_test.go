package jaccount

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/message"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

func TestMarshalEmail(t *testing.T) {
	//this test asserts that the BespokeHeader requests get merged correctly in the know field struct EmailKnownFields

	em := Email{
		EmailDefinedProperties: EmailDefinedProperties{
			EmailMetadata: EmailMetadata{
				Id: basetypes.Id("1"),
			},
		},
		DynamicProperties: map[string]any{
			"h1": "h1value",
		},
		properties: []string{"id"},
	}

	ebpBytes, err := json.Marshal(em)

	if err != nil {
		t.Logf("unexpected error marshalling EmailBodyPart: %s", err)
		t.FailNow()
	}

	expected := []byte(`{"h1":"h1value","id":"1"}`)

	if !bytes.Equal(ebpBytes, expected) {
		t.Logf("was expecting %s but got %s", expected, ebpBytes)
		t.FailNow()
	}
}

func TestMarshalEmailBodyPart(t *testing.T) {
	//this test asserts that the BespokeProperties requests get merged correctly in the know field struct EmailBodyKnowParts

	var name = "name"
	ebp := EmailBodyPart{
		EmailBodyPartKnownFields: EmailBodyPartKnownFields{
			Name: &name,
		},
		BespokeProperties: map[string]any{
			"h1": "h1value",
		},
		properties: []string{"name"},
	}

	ebpBytes, err := json.Marshal(ebp)

	if err != nil {
		t.Logf("unexpected error marshalling EmailBodyPart: %s", err)
		t.FailNow()
	}

	expected := []byte(`{"h1":"h1value","name":"name"}`)

	if !bytes.Equal(ebpBytes, expected) {
		t.Logf("was expecting %s but got %s", expected, ebpBytes)
		t.FailNow()
	}
}

func TestJMAIL(t *testing.T) {

	t.Run("Mail to JEmail", func(t *testing.T) {
		t.Run("Mail to JEmail. Message with no body", func(t *testing.T) {

			mail := `Received: from mail.km42.nl by mail.km42.nl ([46.19.33.172]) via tcp with
        ESMTPSA id E9RpoNC3aqj1FXoMYVjraA (TLS1.3 TLS_AES_128_GCM_SHA256) for
        <jmap@km42.nl>; 18 Jul 2023 17:59:53 +0200
DKIM-Signature: v=1; d=km42.nl; s=2023a; i=me@km42.nl; a=ed25519-sha256;
        t=1689695993; x=1689955193; h=From:To:Cc:Bcc:Reply-To:References:In-Reply-To:
        Subject:Date:Message-Id:Content-Type:From:To:Subject:Date:Message-Id:
        Content-Type; bh=frcCV1k9oG9oKj3dpUqdJg1PxRT2RSN/XKdLCPjaYaY=; b=VLL0zQK7erXE
        /hWH67dxYOF/zMO6JzTVrH9tRTqmP6Wvyju+51eF7ve/f5f8f+rCgnXQmKS7daSphdsoOXIsCQ==
DKIM-Signature: v=1; d=km42.nl; s=2023b; i=me@km42.nl; a=rsa-sha256;
        t=1689695993; x=1689955193; h=From:To:Cc:Bcc:Reply-To:References:In-Reply-To:
        Subject:Date:Message-Id:Content-Type:From:To:Subject:Date:Message-Id:
        Content-Type; bh=frcCV1k9oG9oKj3dpUqdJg1PxRT2RSN/XKdLCPjaYaY=; b=WP2pNyO4GdzE
        Mb7lYhwN3pRcHmiwUBN8Sq9MZQupKNyVwh3UYg0sdR8DJo5R98o5ruv9yOo6+Q89MA635tSahc+m5
        i4i8pebbYYGAamdXQfri4KvfN5qaRlSnKq8P5qsgTQLqZ+3vvJAmG5PknuNd7/Uf271vFejUUynML
        BAsudkQPMyeCaZxvlxLEXdksheN6dy30Z4MOODOQ2ChMvRHHnmCAI+8yfQqShzLQinLzhpi2NfRb0
        S2CennWaFMzEwhTZGTHwWQzkyKvo2HQLIvn7wwUPqGlD6SWntjg85W01HVvUXErKT1V5BZ5ZjmTNs
        XV3ASvYLtFDI+jM1OMVBaA==
Authentication-Results: mail.km42.nl; auth=pass smtp.mailfrom=me@km42.nl
Message-ID: <5a51ce56-387a-1b2e-26bf-133f93c918c1@km42.nl>
Date: Tue, 18 Jul 2023 17:59:42 +0200
MIME-Version: 1.0
User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:102.0) Gecko/20100101
 Thunderbird/102.13.0
Content-Language: en-US
To: jmap@km42.nl
From: me <me@km42.nl>
Subject: first mail
Content-Type: text/plain; charset=UTF-8; format=flowed
Content-Transfer-Encoding: 7bit
`

			mReader := strings.NewReader(strings.ReplaceAll(mail, "\n", "\r\n"))

			part, err := message.Parse(mlog.New("test"), true, mReader)
			RequireNoError(t, err)

			msg := store.Message{
				Received: time.Date(2023, time.July, 18, 17, 59, 53, 0, time.FixedZone("", 2)),
			}
			jem := NewJEmail(msg, part, mlog.New("test"))

			to, mErr := jem.To()
			RequireNoError(t, mErr)

			if len(to) != 1 {
				t.Logf("was expecting one to address but got %d", len(to))
				t.FailNow()
			}
			AssertNotNil(t, to[0].Name)
			AssertEqualString(t, "jmap@km42.nl", to[0].Email)

			from, mErr := jem.From()
			RequireNoError(t, mErr)

			if len(from) != 1 {
				t.Logf("was expecting one to address but got %d", len(to))
				t.FailNow()

			}
			if !(from[0].Name != nil && *from[0].Name == "me" && from[0].Email == "me@km42.nl") {
				t.Logf("unexpected from. From name: %v email: %s", from[0].Name, from[0].Email)
				t.FailNow()
			}

			msgID, mErr := jem.MessagedId()
			t.Helper()
			RequireNoError(t, mErr)

			eMsgID := "5a51ce56-387a-1b2e-26bf-133f93c918c1@km42.nl"
			if len(msgID) != 1 || msgID[0] != eMsgID {
				t.Logf("was expecting %s but %s", eMsgID, msgID)
				t.FailNow()
			}

			subject, mErr := jem.Subject()
			RequireNoError(t, mErr)
			if subject == nil || *subject != "first mail" {
				t.Logf("was expecting subject 'first mail' but got %v", subject)
				t.FailNow()
			}

			eSendAt := "2023-07-18 17:59:42 +0200 CEST"
			sendAt, mErr := jem.SendAt()
			RequireNoError(t, mErr)
			if AssertNotNil(t, sendAt) {
				AssertEqualString(t, eSendAt, time.Time(*sendAt).String())
			}

			eContentType := "text/plain; charset=UTF-8; format=flowed"
			ctIface, mErr := jem.HeaderAs("Content-Type", "asText", false)
			RequireNoError(t, mErr)
			if ct, ok := ctIface.(string); !ok {
				t.Logf("was expecting ctIface to be string but got %T", ctIface)
				t.FailNow()
			} else {
				AssertEqualString(t, eContentType, ct)
			}

			eReceived := msg.Received
			AssertEqualString(t, eReceived.String(), time.Time(jem.ReceivedAt()).String())

			//this an email with no msg body so preview is empty
			ePreview := ""
			preview, mErr := jem.Preview()
			RequireNoError(t, mErr)
			AssertEqualString(t, ePreview, preview)

			bs, mErr := jem.BodyStructure(nil)
			RequireNoError(t, mErr)

			if AssertNotNil(t, bs.Type) {
				AssertEqualString(t, "text/plain", *bs.Type)
			}
			if AssertNotNil(t, bs.Language) {
				//NB: | is an arbitrary token to stringify a string slice to make it comparable
				AssertEqualString(t, strings.Join([]string{"en-US"}, "|"), strings.Join(bs.Language, "|"))
			}
			if AssertNotNil(t, bs.CharSet) {
				//NB: | is an arbitrary token to stringify a string slice to make it comparable
				AssertEqualString(t, "UTF-8", *bs.CharSet)
			}
		})

		t.Run("Mail to JEmail. Message with only text body", func(t *testing.T) {

			mail := `Message-ID: <15f172dc-fe3c-4a6a-941e-707ce6524a73@km42.nl>
Date: Tue, 17 Oct 2023 18:06:06 +0200
MIME-Version: 1.0
User-Agent: Mozilla Thunderbird
Subject: Re: first mail
Content-Language: en-US
To: me <me@km42.nl>
References: <5a51ce56-387a-1b2e-26bf-133f93c918c1@km42.nl>
From: "JMAP@km42.nl" <jmap@km42.nl>
In-Reply-To: <5a51ce56-387a-1b2e-26bf-133f93c918c1@km42.nl>
Content-Type: text/plain; charset=UTF-8; format=flowed
Content-Transfer-Encoding: 7bit

need a mail for testing

On 18-07-2023 17:59, me wrote:
>`

			mReader := strings.NewReader(strings.ReplaceAll(mail, "\n", "\r\n"))

			part, err := message.Parse(mlog.New("test"), true, mReader)
			RequireNoError(t, err)

			msg := store.Message{
				ID:       1,
				Received: time.Date(2023, time.July, 18, 17, 59, 53, 0, time.FixedZone("", 2)),
			}

			jem := NewJEmail(msg, part, mlog.New("test"))

			to, mErr := jem.To()
			RequireNoError(t, mErr)

			if len(to) != 1 {
				t.Logf("was expecting one to address but got %d", len(to))
				t.FailNow()

			}
			if !(*to[0].Name == "me" && to[0].Email == "me@km42.nl") {
				t.Logf("unexpected to. To name: %v email: %s", to[0].Name, to[0].Email)
				t.FailNow()
			}

			inReplyTo, mErr := jem.InReplyTo()
			RequireNoError(t, mErr)
			if len(inReplyTo) != 1 {
				t.Logf("unexpected in reply to. Expected an slice of length 1 but got length %d", len(inReplyTo))
				t.FailNow()
			}
			AssertEqualString(t, "5a51ce56-387a-1b2e-26bf-133f93c918c1@km42.nl", inReplyTo[0])

			references, mErr := jem.References()
			RequireNoError(t, mErr)
			if len(references) != 1 {
				t.Logf("unexpected in references. Expected an slice of length 1 but got length %d", len(inReplyTo))
				t.FailNow()
			}
			AssertEqualString(t, "5a51ce56-387a-1b2e-26bf-133f93c918c1@km42.nl", references[0])

			//Body value no truncating
			bv, mErr := jem.BodyValues(true, false, false, nil)
			RequireNoError(t, mErr)

			if len(bv) != 1 {
				t.Logf("unexpected bodyvalues. Expected an map of size 1 but got size %d", len(bv))
				t.FailNow()
			}
			if body, ok := bv["0"]; !ok {
				t.Log("Expected key 0 in bodyvalues map")
				t.FailNow()
			} else {
				AssertEqualString(t, "need a mail for testing\r\n\r\nOn 18-07-2023 17:59, me wrote:\r\n>", body.Value)
			}

			//Body value with truncating
			maxBytes := basetypes.Uint(4)
			bv2, mErr := jem.BodyValues(true, false, false, &maxBytes)
			RequireNoError(t, mErr)

			if len(bv2) != 1 {
				t.Logf("unexpected bodyvalues. Expected an map of size 1 but got size %d", len(bv2))
				t.FailNow()
			}
			if body, ok := bv2["0"]; !ok {
				t.Log("Expected key 0 in bodyvalues map")
				t.FailNow()
			} else {
				AssertEqualString(t, "need", body.Value)
				AssertTrue(t, body.IsTruncated)
			}

			//BodyValue html
			bvHTML, mErr := jem.BodyValues(false, true, false, &maxBytes)
			RequireNoError(t, mErr)

			if len(bvHTML) != 1 {
				t.Logf("unexpected bodyvalues. Expected an map of size 1 but got size %d", len(bvHTML))
				t.FailNow()
			}
			if body, ok := bvHTML["0"]; !ok {
				t.Log("Expected key 0 in bodyvalues map")
				t.FailNow()
			} else {
				AssertEqualString(t, "need", body.Value)
				AssertTrue(t, body.IsTruncated)
			}

			//FIXME do bodystructure
			bs, mErr := jem.BodyStructure(defaultEmailBodyProperties)
			RequireNoError(t, mErr)

			if AssertNotNil(t, bs.PartId) {
				AssertEqualString(t, "0", *bs.PartId)
			}

			if AssertNotNil(t, bs.BlobId) {
				AssertEqualString(t, "1-0", string(*bs.BlobId))
			}

			if uint(bs.Size) != uint(471) {
				t.Logf("Was expecting size 14 but got %d", bs.Size)
				t.FailNow()
			}

			if numHeaders := len(bs.Headers); numHeaders != 12 {
				t.Logf("Was expecting 12 headers but got %d", numHeaders)
				t.FailNow()
			}

			AssertNil(t, bs.Name)

			if AssertNotNil(t, bs.Type) {
				AssertEqualString(t, "text/plain", *bs.Type)
			}

			if AssertNotNil(t, bs.CharSet) {
				AssertEqualString(t, "UTF-8", *bs.CharSet)
			}

			AssertNil(t, bs.Disposition)
			AssertNil(t, bs.Cid)

			AssertEqualString(t, "en-US", strings.Join(bs.Language, ","))

			AssertNil(t, bs.Location)
			if len(bs.SubParts) != 0 {
				t.Logf("Was expecting 0 subparts but got %d", len(bs.SubParts))
				t.FailNow()
			}

			//Name *string `json:"name"`
			//Type *string `json:"type"`
			//CharSet *string `json:"charSet"`
			//Disposition *string `json:"disposition"`
			//Cid *string `json:"cid"`
			//Language []string `json:"language"`
			//Location *string `json:"location"`

			//If the type is multipart/*, this contains the body parts of each child.
			//SubParts []EmailBodyPart `json:"subParts"`

		})

		t.Run("Mail to JEmail. Mulitpart alternative", func(t *testing.T) {
			mail := `Content-Type: multipart/alternative;
 boundary="------------Z8pBLNP8kO35FOYVOKN5cUf4"
Message-ID: <73720afb-fbad-2feb-1866-12a91cc8defa@km42.nl>
Date: Fri, 6 Oct 2023 19:16:42 +0200
MIME-Version: 1.0
User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:102.0) Gecko/20100101
 Thunderbird/102.15.1
Content-Language: en-US
To: support@mailtemi.com
From: "JMAP@km42.nl" <jmap@km42.nl>
Subject: JMAP issue

This is a multi-part message in MIME format.
--------------Z8pBLNP8kO35FOYVOKN5cUf4
Content-Type: text/plain; charset=UTF-8; format=flowed
Content-Transfer-Encoding: 7bit

Hi,

I am developing a JMAP server Mox (https://github.com/mjl-/mox) and I am 
have some difficulty configuring a JMAP account in the MailTemi ios app.

The steps I follow are:

JMAP -> New Account

I enter email: jmap@km42.nl and the password. Then it says 'Searching 
....' and comes back with "Server Resolve Failed 
(https://mail.km42.nl:443/.well-known.jmap)".

However at the server end, I see a successful retrieval of the session 
object:

Oct 06 19:10:32 mail mox[1403275]: l=debug m="http request" 
cid=18b05e815bd pkg=http httpaccess= handler=jmap method=get 
url=/jmap/session host=mail.km42.nl duration=9.286008ms statuscode=200 pro
to=http/2.0 remoteaddr=83.80.152.96:60721 tlsinfo=tls1.3 
useragent="Mailtemi/1 CFNetwork/1410.0.3 Darwin/22.6.0" referrr= 
size=524 uncompressedsize=1200

Also I see an incoming request for the mailboxes that is properly answered:

Oct 06 19:10:32 mail mox[1403275]: l=debug m="dump http request" 
pkg=jmap payload="POST /jmap/api HTTP/2.0\r\nHost: 
mail.km42.nl\r\nAccept: application/json\r\nAccept-Encoding: gzip, 
deflate, br\
r\nAccept-Language: nl-NL,nl;q=0.9\r\nContent-Length: 
110\r\nContent-Type: application/json\r\nUser-Agent: Mailtemi/1 
CFNetwork/1410.0.3 
Darwin/22.6.0\r\n\r\n{\"methodCalls\":[[\"Mailbox/get\",{\
"accountId\":\"000\",\"ids\":null},\"92cb0\"]],\"using\":[\"urn:ietf:params:jmap:mail\"]}"

Response payload:
Oct 06 19:10:32 mail mox[1403275]: l=debug m="http response" pkg=jmap 
response="{\"methodResponses\":[[\"Mailbox/get\",{\"accountId\":\"000\",\"list\":[{\"id\":\"1\",\"name\":\"Inbox\",\"parentId
\":null,\"role\":\"Inbox\",\"sortOrder\":1,\"totalEmails\":2,\"unreadEmails\":1,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,\"mayRemoveItems\":
true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"2\",\"name\":\"Sent\",\"paren
tId\":null,\"role\":\"Sent\",\"sortOrder\":2,\"totalEmails\":0,\"unreadEmails\":0,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,\"mayRemoveItems\
":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"3\",\"name\":\"Archive\",\"
parentId\":null,\"role\":\"Archive\",\"sortOrder\":3,\"totalEmails\":0,\"unreadEmails\":0,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,\"mayRemo
veItems\":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"4\",\"name\":\"Tras
h\",\"parentId\":null,\"role\":\"Trash\",\"sortOrder\":4,\"totalEmails\":0,\"unreadEmails\":0,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,\"may
RemoveItems\":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"5\",\"name\":\"
Drafts\",\"parentId\":null,\"role\":\"Draft\",\"sortOrder\":5,\"totalEmails\":1,\"unreadEmails\":1,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,
\"mayRemoveItems\":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"6\",\"name
\":\"Junk\",\"parentId\":null,\"role\":\"Junk\",\"sortOrder\":6,\"totalEmails\":0,\"unreadEmails\":0,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":tru
e,\"mayRemoveItems\":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true}],\"notFound\":[],\
"state\":\"stubState\"},\"92cb0\"]],\"sessionState\":\"stubstate\"}"

Response header details:

Oct 06 19:10:32 mail mox[1403275]: l=debug m="http request" 
cid=18b05e815be pkg=http httpaccess= handler=jmap method=post 
url=/jmap/api host=mail.km42.nl duration=1.737832ms statuscode=200 proto=
http/2.0 remoteaddr=83.80.152.96:60722 tlsinfo=tls1.3 
useragent="Mailtemi/1 CFNetwork/1410.0.3 Darwin/22.6.0" referrr= 
size=446 uncompressedsize=2224

This is working in the reference jmap client 
https://jmap.io/jmap-demo-webmail/ but I cannot get it to work with 
Mailtemi.
Is this someting not right at my end or is this not going well at your end?

Looking forward to your response, regards,

A.

--------------Z8pBLNP8kO35FOYVOKN5cUf4
Content-Type: text/html; charset=UTF-8
Content-Transfer-Encoding: 7bit

<html>
  <head>

    <meta http-equiv="content-type" content="text/html; charset=UTF-8">
  </head>
  <body>
    <p>Hi,</p>
    <p>I am developing a JMAP server Mox (<a class="moz-txt-link-freetext" href="https://github.com/mjl-/mox">https://github.com/mjl-/mox</a>)
      and I am have some difficulty configuring a JMAP account in the
      MailTemi ios app.</p>
    <p>The steps I follow are:</p>
    <p>JMAP -&gt; New Account</p>
    <p>I enter email: <a class="moz-txt-link-abbreviated" href="mailto:jmap@km42.nl">jmap@km42.nl</a> and the password. Then it says
      'Searching ....' and comes back with "Server Resolve Failed
      (<a class="moz-txt-link-freetext" href="https://mail.km42.nl:443/.well-known.jmap">https://mail.km42.nl:443/.well-known.jmap</a>)". </p>
    <p>However at the server end, I see a successful retrieval of the
      session object:</p>
    <p><span style="font-family:monospace"><span
          style="color:#000000;background-color:#ffffff;">Oct 06
          19:10:32 mail mox[1403275]: l=debug m="http request"
          cid=18b05e815bd pkg=http httpaccess= handler=jmap method=get
          url=/jmap/session host=mail.km42.nl duration=9.286008ms
          statuscode=200 pro</span><br>
        to=http/2.0 remoteaddr=83.80.152.96:60721 tlsinfo=tls1.3
        useragent="Mailtemi/1 CFNetwork/1410.0.3 Darwin/22.6.0" referrr=
        size=524 uncompressedsize=1200</span></p>
    <p>Also I see an incoming request for the mailboxes that is properly
      answered:</p>
    <p><span style="font-family:monospace"><span
          style="color:#000000;background-color:#ffffff;">Oct 06
          19:10:32 mail mox[1403275]: l=debug m="dump http request"
          pkg=jmap payload="POST /jmap/api HTTP/2.0\r\nHost:
          mail.km42.nl\r\nAccept: application/json\r\nAccept-Encoding:
          gzip, deflate, br\</span><br>
        r\nAccept-Language: nl-NL,nl;q=0.9\r\nContent-Length:
        110\r\nContent-Type: application/json\r\nUser-Agent: Mailtemi/1
        CFNetwork/1410.0.3
        Darwin/22.6.0\r\n\r\n{\"methodCalls\":[[\"Mailbox/get\",{\<br>
"accountId\":\"000\",\"ids\":null},\"92cb0\"]],\"using\":[\"urn:ietf:params:jmap:mail\"]}"</span></p>
    <p>Response payload:<br>
      <span style="font-family:monospace"><span
          style="font-family:monospace"><span
            style="color:#000000;background-color:#ffffff;">Oct 06
            19:10:32 mail mox[1403275]: l=debug m="http response"
            pkg=jmap
response="{\"methodResponses\":[[\"Mailbox/get\",{\"accountId\":\"000\",\"list\":[{\"id\":\"1\",\"name\":\"Inbox\",\"parentId</span><br>
\":null,\"role\":\"Inbox\",\"sortOrder\":1,\"totalEmails\":2,\"unreadEmails\":1,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,\"mayRemoveItems\":<br>
true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"2\",\"name\":\"Sent\",\"paren<br>
tId\":null,\"role\":\"Sent\",\"sortOrder\":2,\"totalEmails\":0,\"unreadEmails\":0,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,\"mayRemoveItems\<br>
":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"3\",\"name\":\"Archive\",\"<br>
parentId\":null,\"role\":\"Archive\",\"sortOrder\":3,\"totalEmails\":0,\"unreadEmails\":0,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,\"mayRemo<br>
veItems\":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"4\",\"name\":\"Tras<br>
h\",\"parentId\":null,\"role\":\"Trash\",\"sortOrder\":4,\"totalEmails\":0,\"unreadEmails\":0,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,\"may<br>
RemoveItems\":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"5\",\"name\":\"<br>
Drafts\",\"parentId\":null,\"role\":\"Draft\",\"sortOrder\":5,\"totalEmails\":1,\"unreadEmails\":1,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":true,<br>
\"mayRemoveItems\":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true},{\"id\":\"6\",\"name<br>
\":\"Junk\",\"parentId\":null,\"role\":\"Junk\",\"sortOrder\":6,\"totalEmails\":0,\"unreadEmails\":0,\"totalThreads\":0,\"unreadThreads\":0,\"myRights\":{\"mayReadItems\":true,\"mayAddItems\":tru<br>
e,\"mayRemoveItems\":true,\"maySetSeen\":true,\"maySetKeywords\":false,\"mayCreateChild\":true,\"mayRename\":true,\"mayDelete\":false,\"maySubmit\":true},\"isSubscribed\":true}],\"notFound\":[],\<br>
"state\":\"stubState\"},\"92cb0\"]],\"sessionState\":\"stubstate\"}"
          <br>
        </span></span></p>
    <p>Response header details:<br>
      <span style="font-family:monospace"><span
          style="font-family:monospace"></span></span></p>
    <p><span style="font-family:monospace"><span
          style="font-family:monospace">Oct 06 19:10:32 mail
          mox[1403275]: l=debug m="http request" cid=18b05e815be
          pkg=http httpaccess= handler=jmap method=post url=/jmap/api
          host=mail.km42.nl duration=1.737832ms statuscode=200 proto=<br>
          http/2.0 remoteaddr=83.80.152.96:60722 tlsinfo=tls1.3
          useragent="Mailtemi/1 CFNetwork/1410.0.3 Darwin/22.6.0"
          referrr= size=446 uncompressedsize=2224<br>
        </span></span></p>
    <p>This is working in the reference jmap client
      <a class="moz-txt-link-freetext" href="https://jmap.io/jmap-demo-webmail/">https://jmap.io/jmap-demo-webmail/</a> but I cannot get it to work
      with Mailtemi. <br>
      Is this someting not right at my end or is this not going well at
      your end?<br>
      <br>
      Looking forward to your response, regards,</p>
    <p>A.<br>
    </p>
  </body>
</html>

--------------Z8pBLNP8kO35FOYVOKN5cUf4--`

			mReader := strings.NewReader(strings.ReplaceAll(mail, "\n", "\r\n"))

			part, err := message.Parse(mlog.New("test"), true, mReader)
			RequireNoError(t, err)

			msg := store.Message{
				ID:       1,
				Received: time.Date(2023, time.July, 18, 17, 59, 53, 0, time.FixedZone("", 2)),
			}

			jem := NewJEmail(msg, part, mlog.New("test"))

			to, mErr := jem.To()
			RequireNoError(t, mErr)

			if len(to) != 1 {
				t.Logf("was expecting one to address but got %d", len(to))
				t.FailNow()
			}
			if to[0].Email != "support@mailtemi.com" {
				t.Logf("unexpected to. To name: %v email: %s", to[0].Name, to[0].Email)
				t.FailNow()
			}

			bodyStructure, mErr := jem.BodyStructure(defaultEmailBodyProperties)
			RequireNoError(t, mErr)

			AssertEqualInt(t, 2, len(bodyStructure.SubParts))

			//FIXME do the bodyvalues part
		})
	})
}

func RequireNoError(t *testing.T, e error) {
	if !(e == nil || reflect.ValueOf(e).IsNil()) {
		t.Helper()
		t.Logf("was expecting no error but got %s", e.Error())
		t.FailNow()
	}
}

func AssertNil(t *testing.T, i any) bool {
	if i == nil || reflect.ValueOf(i).IsNil() {
		return true
	}

	t.Helper()
	t.Logf("was expecting nil but got %+v", i)
	t.Fail()
	return false
}

func AssertNotNil(t *testing.T, i any) bool {
	if i == nil {
		t.Logf("was expecting not nil but nil")
		t.Fail()
		return false
	}
	return true
}

func AssertTrue(t *testing.T, b bool) bool {
	if !b {
		t.Helper()
		t.Logf("was expecting true but got false")
		t.Fail()
	}
	return b
}

func AssertEqualString(t *testing.T, expected, actual string) bool {
	if expected != actual {
		t.Helper()
		t.Logf("was expecting %q but got %q", expected, actual)
		t.Fail()
	}
	return true
}

func AssertEqualInt(t *testing.T, expected, actual int) bool {
	if expected != actual {
		t.Helper()
		t.Logf("was expecting %d but got %d", expected, actual)
		t.Fail()
	}
	return true
}
