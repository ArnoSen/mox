package jaccount

import (
	"bytes"
	"encoding/json"
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

	expected := []byte(`{"h1":"h1value","name":"name","subParts":null}`)

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

A.

--------------Z8pBLNP8kO35FOYVOKN5cUf4
Content-Type: text/html; charset=UTF-8
Content-Transfer-Encoding: 7bit

<html></html>

--------------Z8pBLNP8kO35FOYVOKN5cUf4--`

			mReader := strings.NewReader(strings.ReplaceAll(mail, "\n", "\r\n"))

			part, err := message.Parse(mlog.New("test"), true, mReader)
			RequireNoError(t, err)

			RequireNoError(t, part.Walk(mlog.New("test"), nil))

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

			if AssertNotNil(t, bodyStructure.SubParts[0].Type) {
				AssertEqualString(t, "text/plain", *bodyStructure.SubParts[0].Type)
			}

			if AssertNotNil(t, bodyStructure.SubParts[1].Type) {
				AssertEqualString(t, "text/html", *bodyStructure.SubParts[1].Type)
			}

			//FIXME do the bodyvalues part

			bvTxt, mErr := jem.BodyValues(true, false, false, nil)
			RequireNoError(t, mErr)
			AssertEqualInt(t, 1, len(bvTxt))

			if textValue, ok := bvTxt["0"]; !ok {
				t.Logf("was expecting partId 0 in body values map")
				t.FailNow()
			} else {
				AssertEqualString(t, "Hi,\r\n\r\nA.\r\n", textValue.Value)
			}

			bvHTML, mErr := jem.BodyValues(false, true, false, nil)
			RequireNoError(t, mErr)
			AssertEqualInt(t, 1, len(bvHTML))

			if textValue, ok := bvHTML["1"]; !ok {
				t.Logf("was expecting partId 0 in body values map")
				t.FailNow()
			} else {
				AssertEqualString(t, "<html></html>\r\n", textValue.Value)
			}
		})
	})
}
