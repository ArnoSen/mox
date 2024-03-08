package httphandler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/core"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

type SessionStateStub struct {
	stubstate string
}

func NewSessionStateStub(stubState string) SessionStateStub {
	return SessionStateStub{
		stubstate: stubState,
	}
}

func (sss SessionStateStub) SessionState(ctx context.Context, email string) (string, error) {
	return sss.stubstate, nil
}

func TestAPIHandler(t *testing.T) {

	t.Run("WrongContentType", func(t *testing.T) {

		stubDataType := NewStubDatatype("Test")
		stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

		apiH := NewAPIHandler(core.NewCore(core.CoreCapabilitySettings{}), []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("abc"), "key", nil, mlog.New("test", nil))

		srv := httptest.NewServer(apiH)

		resp, err := srv.Client().Post(srv.URL, "text/html", nil)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected http status code %d but got %d", http.StatusBadRequest, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}
		defer resp.Body.Close()

		eErrorBytes, err := json.Marshal(NewRequestLevelErrorNotJSONContentType())
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if string(eErrorBytes) != string(body) {
			t.Fatalf("expected body '%s' but got '%s'", eErrorBytes, body)
		}

	})
	t.Run("MaxRequestSizeExceeded", func(t *testing.T) {
		stubDataType := NewStubDatatype("Test")
		stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

		coreCapability := core.NewCore(core.CoreCapabilitySettings{
			MaxSizeRequest: 100,
		})

		srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("def"), "key", nil, mlog.New("test", nil)))

		b := bytes.NewBuffer([]byte(strings.Repeat("a", 101)))

		resp, err := srv.Client().Post(srv.URL, "application/json", b)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected http status code %d but got %d", http.StatusBadRequest, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}
		defer resp.Body.Close()

		eErrorBytes, err := json.Marshal(NewRequestLevelErrorCapabilityLimit(LimitTypeMaxSizeRequest, "max request size is 100 bytes"))
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if string(eErrorBytes) != string(body) {
			t.Fatalf("expected body '%s' but got '%s'", eErrorBytes, body)
		}

	})
	t.Run("NotJSON", func(t *testing.T) {
		stubDataType := NewStubDatatype("Test")
		stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

		coreCapability := core.NewCore(core.CoreCapabilitySettings{
			MaxSizeRequest: 100,
		})

		srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("abc"), "key", nil, mlog.New("test", nil)))

		b := bytes.NewBuffer([]byte(strings.Repeat("a", 10)))

		resp, err := srv.Client().Post(srv.URL, "application/json", b)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected http status code %d but got %d", http.StatusBadRequest, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}
		defer resp.Body.Close()

		eErrorBytes, err := json.Marshal(NewRequestLevelErrorNotJSON("invalid character 'a' looking for beginning of value"))
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if string(eErrorBytes) != string(body) {
			t.Fatalf("expected body '%s' but got '%s'", eErrorBytes, body)
		}
	})

	t.Run("NotRequest", func(t *testing.T) {
		t.Run("EmptyJSON", func(t *testing.T) {
			//there is a broad range of json payloads that are not of type request
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest: 100,
			})

			apiH := NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("ttt"), "key", nil, mlog.New("test", nil))

			srv := httptest.NewServer(apiH)

			b := strings.NewReader("{}")
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected http status code %d but got %d", http.StatusBadRequest, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			eErrorBytes, err := json.Marshal(NewRequestLevelErrorNotRequest("'using' empty or no method calls"))
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			if string(eErrorBytes) != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eErrorBytes, body)
			}
		})
		//FIXME for complete coverage more tests are needed here
	})

	t.Run("UnknownCapability", func(t *testing.T) {
		stubDataType := NewStubDatatype("Test")
		stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

		coreCapability := core.NewCore(core.CoreCapabilitySettings{
			MaxSizeRequest: 100,
		})

		srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("iii"), "key", nil, mlog.New("test", nil)))

		b := strings.NewReader(`{ "using": ["urn:nonexisting"], "methodCalls": [ ["method", null, "c1"] ] }`)
		resp, err := srv.Client().Post(srv.URL, "application/json", b)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected http status code %d but got %d", http.StatusBadRequest, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}
		defer resp.Body.Close()

		eErrorBytes, err := json.Marshal(NewRequestLevelErrorUnknownCapability("urn:nonexisting is not a known capability"))
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if string(eErrorBytes) != string(body) {
			t.Fatalf("expected body '%s' but got '%s'", eErrorBytes, body)
		}
	})

	t.Run("UnknownMethod", func(t *testing.T) {
		stubDataType := NewStubDatatype("Test")
		stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

		coreCapability := core.NewCore(core.CoreCapabilitySettings{
			MaxSizeRequest: 100,
		})

		srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("fff"), "key", nil, mlog.New("test", nil)))

		b := strings.NewReader(`{ 
"using": ["urn:test"], 
"methodCalls": [ ["Test/unknown", null, "c1"] ] 
}`)
		resp, err := srv.Client().Post(srv.URL, "application/json", b)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected http status code %d but got %d", http.StatusBadRequest, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("expected no error but got %s", err)
		}
		defer resp.Body.Close()

		//FIXME need to do something with the session state here
		eBody := `{"methodResponses":[[{"error":{"type":"unknownMethod"}},"c1"]],"sessionState":"fff"}`

		if eBody != string(body) {
			t.Fatalf("expected body '%s' but got '%s'", eBody, body)
		}
	})

	stubJaccounter := func() (jaccount.JAccounter, string, *mlevelerrors.MethodLevelError) {
		return NewJAccountStub(), "email", nil
	}

	t.Run("GetMethod", func(t *testing.T) {

		stubAccountOpener := func(log mlog.Log, name string) (*store.Account, error) {
			return &store.Account{}, nil
		}

		t.Run("EmptyAccountID", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest:  200,
				MaxObjectsInGet: 2,
			})

			apiH := NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("bbb"), "key", stubAccountOpener, mlog.New("test", nil)).
				WithOverrideJAccountFactory(stubJaccounter)

			srv := httptest.NewServer(apiH)

			b := strings.NewReader(`{
				"using": ["urn:test"],
				"methodCalls": [ ["Test/get", { "accountId": "","ids": ["id1", "id2"]}, "c1"] ]
				}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"invalidArguments","description":"accountId cannot be empty"}},"c1"]],"sessionState":"bbb"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})

		t.Run("InvalidAccountIDType", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest:  200,
				MaxObjectsInGet: 2,
			})

			srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("lll"), "key", nil, mlog.New("test", nil)).WithOverrideJAccountFactory(stubJaccounter))
			b := strings.NewReader(`{
				"using": ["urn:test"],
				"methodCalls": [ ["Test/get", { "accountId": 123,"ids": ["id1", "id2"]}, "c1"] ]
				}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"invalidArguments","description":"incorrect type for field accountId"}},"c1"]],"sessionState":"lll"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})

		t.Run("Use of both accountId and #accountId", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest:  200,
				MaxObjectsInGet: 2,
			})

			srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("zzz"), "key", nil, mlog.New("test", nil)).WithOverrideJAccountFactory(stubJaccounter))
			b := strings.NewReader(`{
				"using": ["urn:test"],
				"methodCalls": [ ["Test/get", { "accountId": "abc", "#accountId":{ "resultOf":"c1", "name":"Test/get", "path":"/ids" }, "ids": ["id1", "id2"]}, "c1"] ]
				}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"invalidArguments","description":"cannot use 'accountId' and '#accountId' together"}},"c1"]],"sessionState":"zzz"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})

		t.Run("InvalidIdsType", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(
				core.CoreCapabilitySettings{
					MaxSizeRequest:  200,
					MaxObjectsInGet: 2,
				})

			srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("qqq"), "key", nil, mlog.New("test", nil)).WithOverrideJAccountFactory(stubJaccounter))

			b := strings.NewReader(`{
				"using": ["urn:test"],
				"methodCalls": [ ["Test/get", { "accountId": "abc","ids": 1 }, "c1"] ]
				}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"invalidArguments","description":"incorrect type for field ids"}},"c1"]],"sessionState":"qqq"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})

		t.Run("Use of both ids and #ids", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest:  200,
				MaxObjectsInGet: 2,
			})

			srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("hhh"), "key", nil, mlog.New("test", nil)).WithOverrideJAccountFactory(stubJaccounter))

			b := strings.NewReader(`{
				"using": ["urn:test"],
				"methodCalls": [ ["Test/get", { "accountId": "abc","ids": ["id1"], "#ids":{ "path":"/", "resultOf":"c", "name":"Test/get" }}, "c1"] ]
				}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"invalidArguments","description":"cannot use 'ids' and '#ids' together"}},"c1"]],"sessionState":"hhh"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})

		t.Run("InvalidPropertyType", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest:  200,
				MaxObjectsInGet: 2,
			})

			srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("ppp"), "key", nil, mlog.New("test", nil)).WithOverrideJAccountFactory(stubJaccounter))

			b := strings.NewReader(`{
				"using": ["urn:test"],
				"methodCalls": [ ["Test/get", { "accountId": "abc","ids": ["id1"], "properties": [4,5,6]}, "c1"] ]
				}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"invalidArguments","description":"incorrect type for field properties"}},"c1"]],"sessionState":"ppp"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})

		t.Run("Use of both properties and #properties", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest:  400,
				MaxObjectsInGet: 2,
			})

			srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("aaa"), "key", nil, mlog.New("test", nil)).WithOverrideJAccountFactory(stubJaccounter))

			b := strings.NewReader(`{
				"using": ["urn:test"],
				"methodCalls": [ ["Test/get", { "accountId": "abc","ids": ["id1"], "properties": ["4","5","6"], "#properties": {"name":"Test/get","path":"/p1","resultOf":"c1"}}, "c1"] ]
				}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"invalidArguments","description":"cannot use 'properties' and '#properties' together"}},"c1"]],"sessionState":"aaa"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})

		t.Run("RequestTooLargeInGet", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest:  500,
				MaxObjectsInGet: 1,
			})

			srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("bbb"), "key", nil, mlog.New("test", nil)).WithOverrideJAccountFactory(stubJaccounter))

			b := strings.NewReader(`{ 
"using": ["urn:test"], 
"methodCalls": [ ["Test/get", { "accountId": "abc" ,"ids": ["id1", "id2"]}, "c1"] ] 
}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"requestTooLarge"}},"c1"]],"sessionState":"bbb"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})
	})

	t.Run("SetMethod", func(t *testing.T) {
		t.Run("RequestTooLargeInSet", func(t *testing.T) {
			stubDataType := NewStubDatatype("Test")
			stubCapability := NewStubCapacility("urn:test", nil, stubDataType)

			coreCapability := core.NewCore(core.CoreCapabilitySettings{
				MaxSizeRequest:  200,
				MaxObjectsInSet: 1,
			})

			srv := httptest.NewServer(NewAPIHandler(coreCapability, []capabilitier.Capabilitier{stubCapability}, NewSessionStateStub("zyx"), "key", nil, mlog.New("test", nil)).WithOverrideJAccountFactory(stubJaccounter))

			b := strings.NewReader(`{ 
"using": ["urn:test"], 
"methodCalls": [ ["Test/set", { "create": {"id1": null}, "update": {"id2": null }}, "c1"] ] 
}`)
			resp, err := srv.Client().Post(srv.URL, "application/json", b)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}

			eStatus := http.StatusOK

			if resp.StatusCode != eStatus {
				t.Fatalf("expected http status code %d but got %d", eStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("expected no error but got %s", err)
			}
			defer resp.Body.Close()

			//FIXME need to do something with the session state here
			eBody := `{"methodResponses":[[{"error":{"type":"requestTooLarge"}},"c1"]],"sessionState":"zyx"}`

			if eBody != string(body) {
				t.Fatalf("expected body '%s' but got '%s'", eBody, body)
			}
		})
	})
}

func TestResolveJSONPointer(t *testing.T) {

	const exampleJSONRFC6901 = `{ "foo": ["bar", "baz"], "": 0, "a/b": 1, "c%d": 2, "e^f": 3, "g|h": 4, "i\\j": 5, "k\"l": 6, " ": 7, "m~n": 8 }`

	for _, testcase := range []struct {
		TestName string
		JSON     string
		Pointer  string
		EResult  json.RawMessage
		EError   bool
	}{
		{
			TestName: "RFC6901Example1",
			JSON:     exampleJSONRFC6901,
			Pointer:  "",
			EResult:  json.RawMessage([]byte(`{"":0," ":7,"a/b":1,"c%d":2,"e^f":3,"foo":["bar","baz"],"g|h":4,"i\\j":5,"k\"l":6,"m~n":8}`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example2",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/foo",
			EResult:  json.RawMessage([]byte(`["bar","baz"]`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example3",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/foo/0",
			EResult:  json.RawMessage([]byte(`"bar"`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example4",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/",
			EResult:  json.RawMessage([]byte(`0`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example5",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/a~1b",
			EResult:  json.RawMessage([]byte(`1`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example6",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/c%d",
			EResult:  json.RawMessage([]byte(`2`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example7",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/e^f",
			EResult:  json.RawMessage([]byte(`3`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example8",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/g|h",
			EResult:  json.RawMessage([]byte(`4`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example9",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/i\\j",
			EResult:  json.RawMessage([]byte(`5`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example10",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/k\"l",
			EResult:  json.RawMessage([]byte(`6`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example11",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/ ",
			EResult:  json.RawMessage([]byte(`7`)),
			EError:   false,
		},
		{
			TestName: "RFC6901Example12",
			JSON:     exampleJSONRFC6901,
			Pointer:  "/m~0n",
			EResult:  json.RawMessage([]byte(`8`)),
			EError:   false,
		},
		{
			TestName: "EmptyPointer",
			JSON:     `{ "id": "a" }`,
			Pointer:  "",
			EResult:  json.RawMessage([]byte(`{"id":"a"}`)),
			EError:   false,
		},
		{
			TestName: "SingleElementJSON",
			JSON:     `{ "id": "a" }`,
			Pointer:  "/id",
			EResult:  json.RawMessage([]byte(`"a"`)),
			EError:   false,
		},
		{
			TestName: "SingleArrayJSON",
			JSON:     `{ "ids": ["a", "b"] }`,
			Pointer:  "/ids/0",
			EResult:  json.RawMessage([]byte(`"a"`)),
			EError:   false,
		},
		{
			TestName: "ArrayAtSecondLevel",
			JSON:     `{ "some": { "ids": ["a", "b"] } }`,
			Pointer:  "/some/ids/0",
			EResult:  json.RawMessage([]byte(`"a"`)),
			EError:   false,
		},
		{
			TestName: "PropertyInArrayOfObjects",
			JSON:     `{ "ids": [ { "a": 1 }, {"b": 2} ] }`,
			Pointer:  "/ids/0/a",
			EResult:  json.RawMessage(string("1")),
			EError:   false,
		},
		{
			TestName: "ReturnArray",
			JSON:     `{ "ids": [ { "a": 1 }, {"b": 2} ] }`,
			Pointer:  "/ids",
			EResult:  json.RawMessage(string(`[{"a":1},{"b":2}]`)),
			EError:   false,
		},
		{
			TestName: "SpecialCharAsterisk",
			JSON:     `{ "some": { "ids": [ { "a": "foo" }, {"a": "bar"} ]  }}`,
			Pointer:  "/some/ids/*/a",
			EResult:  json.RawMessage(string(`["foo","bar"]`)),
			EError:   false,
		},
		{
			TestName: "SpecialCharAsteriskWithFlatten",
			JSON:     `{ "some": { "ids": [ { "a": ["foo"] }, {"a": ["bar"]} ]  }}`,
			Pointer:  "/some/ids/*/a",
			EResult:  json.RawMessage(string(`["foo","bar"]`)),
			EError:   false,
		},
		{
			TestName: "Real jmap web client example",
			JSON:     `{"accountId":"000","list":[{"id":"2","emailIds":["2"]},{"id":"1","emailIds":["1"]}],"notFound":null,"state":"stubstate"}`,
			Pointer:  "/list/*/emailIds",
			EResult:  json.RawMessage(string(`["2","1"]`)),
			EError:   false,
		},
		{
			TestName: "QueryEmptySet",
			JSON:     `{"accountId":"000","list":[],"notFound":null,"state":"stubstate"}`,
			Pointer:  "/list/*/emailIds",
			//FIXME not sure if this is the output we need
			EResult: json.RawMessage(string(`null`)),
			EError:  false,
		},
	} {

		t.Run(testcase.TestName, func(t *testing.T) {
			result, err := resolveJSONPointer(json.RawMessage(testcase.JSON), testcase.Pointer)
			if err != nil {
				if !testcase.EError {
					t.Fatalf("was not expecting an error but got: %s", err)
				} //test passed
			} else {
				if testcase.EError {
					t.Fatalf("was expecting an error but ok")
				} else {
					if string(result) != string(testcase.EResult) {
						t.Fatalf("was expecting '%s' but got '%s'", testcase.EResult, result)
					}
				}
			}
		})
	}
}

func TestFilterProperties(t *testing.T) {

	t.Run("RemoveSomeProperties", func(t *testing.T) {
		type obj struct {
			A string `json:"a"`
			B string `json:"b"`
			C string `json:"c"`
			Z string `json:"z"`
		}

		myObjs := []interface{}{
			obj{
				A: "a",
				B: "b",
				C: "c",
				Z: "z",
			},
			obj{
				A: "d",
				B: "e",
				C: "f",
				Z: "z",
			},
		}

		filteredObjs, err := filterProperties(myObjs, []string{"a", "c"}, []string{"z"})
		if err != nil {
			t.Fatalf("was not expecting an err but got '%s'", err)
		}

		for _, filteredObj := range filteredObjs {
			mapStringIface, ok := filteredObj.(map[string]interface{})
			if !ok {
				t.Fatalf("was expecting type map[string]interface{} but got '%T'", filteredObj)
			}
			if _, found := mapStringIface["a"]; !found {
				t.Fatalf("was expecting to find a property named 'a'")
			}
			if _, found := mapStringIface["c"]; !found {
				t.Fatalf("was expecting to find a property named 'c'")
			}
			if _, found := mapStringIface["z"]; !found {
				t.Fatalf("was expecting to find a property named 'z'")
			}
			if _, found := mapStringIface["b"]; found {
				t.Fatalf("was not expecting to find a property named 'b'")
			}
		}
	})

	t.Run("Empty properties returns the object without modifications", func(t *testing.T) {
		type obj struct {
			A string `json:"a"`
			B string `json:"b"`
			C string `json:"c"`
			Z string `json:"z"`
		}

		myObjs := []interface{}{
			obj{
				A: "a",
				B: "b",
				C: "c",
				Z: "z",
			},
		}

		filteredObjs, err := filterProperties(myObjs, []string{}, []string{"z"})
		if err != nil {
			t.Fatalf("was not expecting an err but got '%s'", err)
		}

		for i, filteredObj := range filteredObjs {

			myObj, ok := filteredObj.(obj)
			if !ok {
				t.Fatalf("was expecting type  but got '%T'", filteredObj)
			}
			if myObj != myObjs[i] {
				t.Fatalf("was expecting to find object not modified but got %+v", filteredObj)
			}
		}
	})
}
