package jaccount

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

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

func TestPartToEmailBodyPart(t *testing.T) {

	t.Run("MailX", func(t *testing.T) {

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

		jem := NewJEmail(store.Message{}, part, mlog.New("test"))

		to, mErr := jem.To()
		RequireNoError(t, mErr)

		if len(to) != 1 {
			t.Logf("was expecting one to address but got %d", len(to))
			t.FailNow()

			if !(to[0].Name == nil && to[0].Email == "jmap@km42.nl") {
				t.Logf("unexpected to. To name: %v email: %s", to[0].Name, to[0].Email)
				t.FailNow()
			}
		}

		from, mErr := jem.From()
		RequireNoError(t, mErr)

		if len(from) != 1 {
			t.Logf("was expecting one to address but got %d", len(to))
			t.FailNow()

			if !(to[0].Name != nil && *to[0].Name == "me" && to[0].Email == "me@km42.nl") {
				t.Logf("unexpected from. From name: %v email: %s", to[0].Name, to[0].Email)
				t.FailNow()
			}
		}

		msgID, mErr := jem.MessagedId()
		RequireNoError(t, mErr)

		eMsgID := "5a51ce56-387a-1b2e-26bf-133f93c918c1@km42.nl"
		if len(msgID) != 1 || msgID[0] != eMsgID {
			t.Logf("was expecting %s but %s", eMsgID, msgID)
			t.FailNow()
		}

		subject, mErr := jem.Subject()
		RequireNoError(t, mErr)
		if !(subject == nil || *subject != "first mail") {
			t.Logf("was expecting subject 'first mail' but got %v", subject)
			t.FailNow()
		}
	})
}

func RequireNoError(t *testing.T, e error) {
	if !(e == nil || reflect.ValueOf(e).IsNil()) {
		t.Logf("was expecting no error but got %s", e.Error())
		t.FailNow()
	}
}
