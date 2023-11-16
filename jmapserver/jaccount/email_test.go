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
				AssertEqualString(t, "text/plain", bs.Type.String())
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
				AssertEqualString(t, "text/plain", bs.Type.String())
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
				AssertEqualString(t, "text/plain", bodyStructure.SubParts[0].Type.String())
			}

			if AssertNotNil(t, bodyStructure.SubParts[1].Type) {
				AssertEqualString(t, "text/html", bodyStructure.SubParts[1].Type.String())
			}

			/*
				bvTxt, mErr := jem.BodyValues(true, false, false, nil)
				RequireNoError(t, mErr)
				AssertEqualInt(t, 1, len(bvTxt))

					if textValue, ok := bvTxt["0"]; !ok {
						t.Logf("was expecting partId 0 in body values map")
						t.FailNow()
					} else {
						AssertEqualString(t, "Hi,\r\n\r\nA.\r\n", textValue.Value)
					}
			*/

			bvHTML, mErr := jem.BodyValues(false, true, false, nil)
			RequireNoError(t, mErr)
			AssertEqualInt(t, 1, len(bvHTML))

			if textValue, ok := bvHTML["1"]; !ok {
				t.Logf("was expecting partId 1 in body values map")
				t.FailNow()
			} else {
				AssertEqualString(t, "<html></html>\r\n", textValue.Value)
			}
		})

		t.Run("Mail to JEmail. Inline picture", func(t *testing.T) {
			mail := `X-Mox-Reason: msgtofull
Delivered-To: jmap@km42.nl
Return-Path: <me@km42.nl>
Authentication-Results: mail.km42.nl; iprev=pass (without dnssec)
	policy.iprev=2a02:2770::21a:4aff:fe09:2980; dkim=pass (without dnssec)
	header.d=km42.nl header.s=2023a header.a=ed25519-sha256 header.b=MloqXOsyamEz
	header.i=me@km42.nl; dkim=pass (2048 bit rsa, without dnssec)
	header.d=km42.nl header.s=2023b header.a=rsa-sha256 header.b=YRfIQ8JhSMMK
	header.i=me@km42.nl; spf=pass (without dnssec) smtp.mailfrom=km42.nl;
	dmarc=pass (without dnssec) header.from=km42.nl
Received-SPF: pass (domain km42.nl) client-ip="2a02:2770::21a:4aff:fe09:2980";
	envelope-from="me@km42.nl"; helo=mail.km42.nl;
	mechanism="ip6:2a02:2770::21a:4aff:fe09:2980"; receiver=mail.km42.nl;
	identity=mailfrom
Received: from mail.km42.nl ([IPv6:2a02:2770::21a:4aff:fe09:2980]) by
	mail.km42.nl ([IPv6:2a02:2770::21a:4aff:fe09:2980]) via tcp with ESMTPS id
	7_jO3KP4COnlTQm3SYtefQ (TLS1.3 TLS_AES_128_GCM_SHA256) for <jmap@km42.nl>;
	15 Nov 2023 21:38:59 +0100
Received: from mail.km42.nl by mail.km42.nl ([46.19.33.172]) via tcp with
	ESMTPSA id Kn4f0j26XW4HmrhWCoWxTQ (TLS1.3 TLS_AES_128_GCM_SHA256) for
	<jmap@km42.nl>; 15 Nov 2023 21:38:59 +0100
DKIM-Signature: v=1; d=km42.nl; s=2023a; i=me@km42.nl; a=ed25519-sha256;
	t=1700080739; x=1700339939; h=From:To:Cc:Bcc:Reply-To:References:In-Reply-To:
	Subject:Date:Message-Id:Content-Type:From:To:Subject:Date:Message-Id:
	Content-Type; bh=uVgXySeyW0cQ+CvbpukhA0uP6zzMx5KKnL3ZA5QkYlc=; b=MloqXOsyamEz
	sP1yAvisewIn0PI+FY5Dhcznk8XzyiBJfdjYWX0hvaUI8fYBb54ddJPlIf0ANuo2kNZqW5ImBw==
DKIM-Signature: v=1; d=km42.nl; s=2023b; i=me@km42.nl; a=rsa-sha256;
	t=1700080739; x=1700339939; h=From:To:Cc:Bcc:Reply-To:References:In-Reply-To:
	Subject:Date:Message-Id:Content-Type:From:To:Subject:Date:Message-Id:
	Content-Type; bh=uVgXySeyW0cQ+CvbpukhA0uP6zzMx5KKnL3ZA5QkYlc=; b=YRfIQ8JhSMMK
	iGhp19P8GYMtbq5YWQpzxnha7Pr0K4ayc2bsmA4ZKYdJdjD6ESOeuyoIc2ohXH+b631zap6n+mki9
	Gn1PvIMqe4LUyIEHSEpBSJFsF63kmIpQvUMyoF95x/yy4T3X+//4KgsewZXedgX7SV1rLEBc3Q7kF
	0gpQ3L4omNpYgbYAItagnq3hjGwPwvgDtR2DHqwIFhGCXNclOdSABtB8VbQUvC13IksjpJdNB8bX/
	lSl6GWSFhP81DcXnSo/AXO5ceTQ+ibPJfkirrFA7E1iQaGZOzDreGJlTIpRr5D9y1QtY0o8bpzwHT
	GqzNIhvaSIUxjEi/syFVEQ==
Authentication-Results: mail.km42.nl; auth=pass smtp.mailfrom=me@km42.nl
Content-Type: multipart/alternative;
 boundary="------------70p0smUx9red4W60tXQ0HJyx"
Message-ID: <ae7a6a03-df9e-47c8-a6c4-a3dd6ff33599@km42.nl>
Date: Wed, 15 Nov 2023 21:38:58 +0100
MIME-Version: 1.0
User-Agent: Mozilla Thunderbird
Content-Language: en-US
To: jmap@km42.nl
From: me <me@km42.nl>
Subject: image

This is a multi-part message in MIME format.
--------------70p0smUx9red4W60tXQ0HJyx
Content-Type: text/plain; charset=UTF-8; format=flowed
Content-Transfer-Encoding: 7bit

My first image

--------------70p0smUx9red4W60tXQ0HJyx
Content-Type: multipart/related;
 boundary="------------sSdEUDikeN4cbn6FvgeAoU0v"

--------------sSdEUDikeN4cbn6FvgeAoU0v
Content-Type: text/html; charset=UTF-8
Content-Transfer-Encoding: 7bit

<!DOCTYPE html>
<html>
  <head>

    <meta http-equiv="content-type" content="text/html; charset=UTF-8">
  </head>
  <body>
    <p>My first image <img src="cid:part1.Nj2N9maO.uVlYYEhk@km42.nl"
        alt=""></p>
  </body>
</html>
--------------sSdEUDikeN4cbn6FvgeAoU0v
Content-Type: image/png; name="kOp2KOEom97WsgRN.png"
Content-Disposition: inline; filename="kOp2KOEom97WsgRN.png"
Content-Id: <part1.Nj2N9maO.uVlYYEhk@km42.nl>
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAACEAAAAhCAIAAADYhlU4AAAACXBIWXMAABYlAAAWJQFJUiTw
AAABbklEQVRIie3WsVOCUBwH8J9dk88xaJTnCK3A7ixzruGsa/oX5BvMu+o/cIG2imtpbHp4
TT5aofU5Zfy8thq8c2gQVGwovxtf+N3nHhzwSu+zD9hxDnYN7I1/aRzmvC5FfI0iOZW6btQo
Ld6YCMEYQ8TFoWVbvfNufiP7XqWIjDFFORoMBqPRyHXPQh56vlekEXKOiN1ur0ZphRCn4ZiW
GYZhkYacSgA4VtVlU6M0jpMije2zhpEics5XN9saQfDQZ2x1s62xcfbG3thF8n7bAUBVVAC4
YP1IRJpWzT33ucb/o16vx0kSBIGiqu12BwAQ55mDX7PnbINqFAAiIWzbbrluy3WXp4SYZC5o
+nSX/Txs2y6XiX/r/+g550ny5jSc1eOP9y+lPPtEznmfMUq15mlTNwwp5Xgcep6vadXh5TBz
PJexYK6ub+ZzXDamZXbanQohhRkAkCJGQsRJTAgxjJP8O4c1jI3zV97z3zC+ASthnSqsAY+j
AAAAAElFTkSuQmCC

--------------sSdEUDikeN4cbn6FvgeAoU0v--

--------------70p0smUx9red4W60tXQ0HJyx--
`

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
			if to[0].Email != "jmap@km42.nl" {
				t.Logf("unexpected to. To name: %v email: %s", to[0].Name, to[0].Email)
				t.FailNow()
			}

			bs, merr := jem.BodyStructure(defaultEmailBodyProperties)
			RequireNoError(t, merr)

			AssertEqualInt(t, 2, len(bs.SubParts))

			AssertEqualInt(t, 2, len(bs.SubParts[1].SubParts))

			imgPart := bs.SubParts[1].SubParts[1]
			AssertEqualString(t, "image/png", imgPart.Type.String())
			AssertEqualString(t, "part1.Nj2N9maO.uVlYYEhk@km42.nl", *imgPart.Cid)
			AssertEqualString(t, "kOp2KOEom97WsgRN.png", *imgPart.Name)
			AssertEqualString(t, "2", *imgPart.PartId)

			bv, mErr := jem.BodyValues(false, true, false, nil)
			RequireNoError(t, mErr)
			AssertEqualInt(t, 1, len(bv))
			htmlBodyValue, ok := bv["1"]
			AssertTrue(t, ok)
			AssertEqualString(t, "<!DOCTYPE html>\r\n<html>\r\n  <head>\r\n\r\n    <meta http-equiv=\"content-type\" content=\"text/html; charset=UTF-8\">\r\n  </head>\r\n  <body>\r\n    <p>My first image <img src=\"cid:part1.Nj2N9maO.uVlYYEhk@km42.nl\"\r\n        alt=\"\"></p>\r\n  </body>\r\n</html>", htmlBodyValue.Value)

			htmlBody, mErr := jem.HTMLBody(defaultEmailBodyProperties)
			RequireNoError(t, mErr)
			AssertEqualInt(t, 2, len(htmlBody))
		})

		t.Run("Mail to JEmail. Picture as attachemnt. No html available", func(t *testing.T) {
			mail := `X-Mox-Reason: no-bad-signals
Delivered-To: jmap@km42.nl
Return-Path: <arno.overgaauw@mailbox.org>
Authentication-Results: mail.km42.nl; iprev=pass (without dnssec)
	policy.iprev=2001:67c:2050:0:465::103; dkim=pass
	(2048 bit rsa, without dnssec) header.d=mailbox.org header.s=mail20150812
	header.a=rsa-sha256 header.b=ZySjhVQq20fa; spf=pass (without dnssec) smtp.mailfrom=mailbox.org; dmarc=pass (without dnssec)
	header.from=mailbox.org
Received-SPF: pass (domain mailbox.org) client-ip="2001:67c:2050:0:465::103";
	envelope-from="arno.overgaauw@mailbox.org"; helo=mout-p-103.mailbox.org;
	mechanism="ip6:2001:67c:2050::/48"; receiver=mail.km42.nl; identity=mailfrom
Received: from mout-p-103.mailbox.org ([IPv6:2001:67c:2050:0:465::103]) by
	mail.km42.nl ([IPv6:2a02:2770::21a:4aff:fe09:2980]) via tcp with ESMTPS id
	HDUnJVspeTh-eTnzPmFomQ (TLS1.3 TLS_AES_128_GCM_SHA256) for <jmap@km42.nl>;
	16 Nov 2023 00:46:02 +0100
Received: from smtp2.mailbox.org (smtp2.mailbox.org [IPv6:2001:67c:2050:b231:465::2])
	(using TLSv1.3 with cipher TLS_AES_256_GCM_SHA384 (256/256 bits)
	 key-exchange X25519 server-signature RSA-PSS (4096 bits) server-digest SHA256)
	(No client certificate requested)
	by mout-p-103.mailbox.org (Postfix) with ESMTPS id 4SW0D73vfqz9skp
	for <jmap@km42.nl>; Thu, 16 Nov 2023 00:45:59 +0100 (CET)
DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed; d=mailbox.org; s=mail20150812;
	t=1700091959;
	h=from:from:reply-to:subject:subject:date:date:message-id:message-id:
	 to:to:cc:mime-version:mime-version:content-type:content-type:
	 content-transfer-encoding:content-transfer-encoding;
	bh=cv+PD0yMNTODhvrvvv2jP983OT7lFjRK7HP51VzcBFg=;
	b=ZySjhVQq20fahe5Bh5aTj5UFwXrybE7BIP7s7dNDGSxVNazgD4zMlcz9lbS3p+lzylA6UO
	jv3PQWF8bReGOP0t/CWgRge7BlqvztaA46eTatnQ7NQmIWkDOJWiGrB1v41gGr9QtorRnx
	e7svIIsjpF+L8HQOyoWn7mIrSoL783ncfJz9bDGlVE/ASfm/auYjdV+H2aSPtC760x4FSG
	UWmHjr8y6/8Vme0It7WImQ8BumN5+oZXJDnv0x6AjtT0kCqMfENBTYV1UYMLuptv9XGrH2
	brFtnxDkYph7gMqBwE8t02agATKogHnOTJHh8910rdSUfq8GUTPG2DXmaYQanA==
Content-Type: multipart/mixed; boundary=Apple-Mail-C592FF4B-B8D0-46D3-BEBB-B8200C3105AA
Content-Transfer-Encoding: 7bit
From: Arno Overgaauw <arno.overgaauw@mailbox.org>
Mime-Version: 1.0 (1.0)
Date: Thu, 16 Nov 2023 00:45:57 +0100
Subject: Test2
Message-Id: <0667EA89-1A31-4740-8453-89A2699EDA21@mailbox.org>
To: jmap@km42.nl
X-MBO-RS-ID: a053518dbb361b91200
X-MBO-RS-META: cxmncpms3ofwi5u8b9wytbxfdqwb59dp


--Apple-Mail-C592FF4B-B8D0-46D3-BEBB-B8200C3105AA
Content-Type: text/plain;
	charset=us-ascii
Content-Transfer-Encoding: 7bit




--Apple-Mail-C592FF4B-B8D0-46D3-BEBB-B8200C3105AA
Content-Type: image/png;
	name=IMG_0361.PNG;
	x-apple-part-url=DE3DAE08-C43E-4537-A42F-F32617D93C6D
Content-Disposition: inline;
	filename=IMG_0361.PNG
Content-Transfer-Encoding: base64

iVBORw0KGgoAAAANSUhEUgAAALQAAAFAEAIAAADKSx2dAAAAAXNSR0IArs4c6QAAAI5lWElmTU0A
KgAAAAgAAgESAAMAAAABAAEAAIdpAAQAAAABAAAAJgAAAAAABZADAAIAAAAUAAAAaJKGAAcAAAAS
AAAAfKABAAMAAAABAAEAAKACAAQAAAABAAAAtKADAAQAAAABAAABQAAAAAAyMDIzOjExOjEyIDEy
RVZ8FVJzaZKbmi1iEK3SU8Cx86ggoTigWSSCfsyau/IQWry/lNh+LW8fCgPWS6GGWWQPRYGVGCSH
nOyqEPF33Kr5Dxki/AZvMotqA6MkUxHkVRyvoUfjGNn8UCBZNqKk2UQvO+gx/pKUd9/M0BZl1kE8
UcLPiOxzRnaYb8VDtz03D0SKp0dw7QCwFIl6NDOZlBrB2y0l0/661ABaZKl8Rtl00pYKKKXbmXUN
p5RzbKngn+oSFbMHljoQ9Ea2NIq6NCFbj9GvrgJd5TVXmtRSnEE2RzoFxnAcioWkOZIWi/il+GQp
MUS3GiWeINu6Suppwpwf4VcxAfd3ac66HM/gH6Wsi7wkyBSdyLJgbMyPTgA6ntfSwomkkCwkJbdo
xQ5GUMwL8y7jq5euKqOlxB3raKF5YGr4mZBt4QhGUNuWYGmGFnq6nIB1HSvL2K9JrJGu/wE+vnrF
5la42AAAAABJRU5ErkJggg==

--Apple-Mail-C592FF4B-B8D0-46D3-BEBB-B8200C3105AA
Content-Type: text/plain;
	charset=us-ascii
Content-Transfer-Encoding: 7bit



Sent from mobile
--Apple-Mail-C592FF4B-B8D0-46D3-BEBB-B8200C3105AA--`
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
			if to[0].Email != "jmap@km42.nl" {
				t.Logf("unexpected to. To name: %v email: %s", to[0].Name, to[0].Email)
				t.FailNow()
			}

			bs, merr := jem.BodyStructure(defaultEmailBodyProperties)
			RequireNoError(t, merr)

			AssertEqualInt(t, 3, len(bs.SubParts))

			textBodyParts, merr := jem.TextBody(defaultEmailBodyProperties)
			RequireNoError(t, merr)
			AssertEqualInt(t, 3, len(textBodyParts))
			AssertEqualString(t, "0", *textBodyParts[0].PartId)
			AssertEqualString(t, "1", *textBodyParts[1].PartId)
			AssertEqualString(t, "2", *textBodyParts[2].PartId)

			hasAttachment, merr := jem.HasAttachment()
			RequireNoError(t, merr)
			AssertTrue(t, !hasAttachment)
		})
	})
}
