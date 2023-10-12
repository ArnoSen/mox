package jaccount

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mjl-/mox/jmapserver/basetypes"
)

func TestMarshalEmail(t *testing.T) {

	em := Email{
		EmailKnownFields: EmailKnownFields{
			EmailMetadata: EmailMetadata{
				Id: basetypes.Id("1"),
			},
		},
		BespokeHeaderRequests: map[string]any{
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
