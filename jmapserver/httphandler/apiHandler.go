package httphandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"

	"log/slog"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/datatyper"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/jmapserver/user"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

const (
	HeaderContentType         = "Content-Type"
	HeaderContentTypeJSON     = "application/json"
	HeaderContentTypeJSONUTF8 = "application/json;charset=utf-8"
)

// Request is the top level request object for the api handler
type Request struct {
	//Using contains the set of capabilities the client wishes to use
	Using []string `json:"using"`

	//MethodCalls is an array of method calls to process on the server
	MethodCalls []InvocationRequest `json:"methodCalls"`

	//CreatedIds is an  (optional) map of a (client-specified) creation id to the id the server assigned when a record was successfully created.
	CreatedIds map[basetypes.Id]basetypes.Id `json:"createdIds"`
}

// InvocationRequest is a call to datatype's method
// NB: there are no JSON tags here. This is handled in the custom umarshaler
type InvocationRequest struct {
	Name         string
	Arguments    json.RawMessage
	MethodCallID string
}

func (inv *InvocationRequest) UnmarshalJSON(b []byte) error {
	/*
	   Invocation consists of 3 elements:
	   1. name (string) (Format:  <datatype>/[get|changes|set|copy|query|querychanges])
	   2. arguments (map[string]interface{})
	   3. method call id (string)
	*/
	type invocationTuple [3]json.RawMessage

	var it invocationTuple

	if err := json.Unmarshal(b, &it); err != nil {
		switch e := err.(type) {
		case *json.InvalidUnmarshalError:
			//InvalidUnmarshalError is only returned when a non pointer is provided to Decode/Unmarshal
			return e
		case *json.SyntaxError:
			//SyntaxError means the JSON is invalid
			return NewRequestLevelErrorNotJSON(err.Error())
		case *json.UnmarshalTypeError:
			return NewRequestLevelErrorNotRequest(fmt.Sprintf("error in %s", e.Field))
		default:
			return e
		}
	}

	//parse the name
	var name string
	if err := json.Unmarshal(it[0], &name); err != nil {
		return NewRequestLevelErrorNotRequest("invocation name must be a string")
	}

	var commandReference string
	if err := json.Unmarshal(it[2], &commandReference); err != nil {
		return NewRequestLevelErrorNotRequest("command reference name must be a string")
	}

	*inv = InvocationRequest{
		Name:         name,
		Arguments:    it[1],
		MethodCallID: commandReference,
	}

	return nil
}

// InvocationResponse is of type spec.Invocation but with slightly different types as InvocationRequest
// NB: there are no JSON tags because this is marshalled into a tuple
type InvocationResponse struct {
	//Name is not returned when invocation is used in the reponse and is an error
	Name         string
	Arguments    map[string]interface{}
	MethodCallID string
}

// MarshalJSON is a custommarshaller because we need to return a tuple here
func (invResp InvocationResponse) MarshalJSON() ([]byte, error) {
	var resp []interface{}

	if _, isError := invResp.Arguments["error"]; isError {
		resp = append(resp, invResp.Arguments, invResp.MethodCallID)
	} else {
		resp = append(resp, invResp.Name, invResp.Arguments, invResp.MethodCallID)
	}

	return json.Marshal(resp)
}

// newInvocationResponse instantiates a new empty reponse with only the methodCallID set
func newInvocationResponse(methodCallID string) InvocationResponse {
	return InvocationResponse{
		MethodCallID: methodCallID,
	}
}

// withArgError adds an error to a invocation response
func (inv InvocationResponse) withArgError(mErr *mlevelerrors.MethodLevelError) InvocationResponse {
	inv.Arguments = map[string]interface{}{
		"error": mErr,
	}
	return inv
}

// withArgError adds a method output to a invocation reponse
func (inv InvocationResponse) withArgOK(methodCall string, args map[string]interface{}) InvocationResponse {
	inv.Arguments = args
	inv.Name = methodCall
	return inv
}

// Response is the top level response that is sent by the API handler
type Response struct {
	MethodResponses []InvocationResponse `json:"methodResponses"`
	CreatedIds      []basetypes.Id       `json:"createdIds,omitempty"`
	SessionState    string               `json:"sessionState"`
}

// getResultByRef resolves the ResultReference
func (r Response) getResultByRef(logger mlog.Log, resultRef *ResultReference, anchorName string, unmarshalAs any) *mlevelerrors.MethodLevelError {
	for _, resp := range r.MethodResponses {
		if resp.MethodCallID == resultRef.ResultOf {
			//need to check if the name of the method matches
			if resp.Name != resultRef.Name {
				//FIXME this will be triggered when in a chain of references an intermediate method fails
				logger.Error("method name is not matching with method call id", slog.Any("resp.Name", resp.Name), slog.Any("resultRef.Name", resultRef.Name))
				return mlevelerrors.NewMethodLevelErrorInvalidResultReference("method name is not matching with method call id")
			}

			//we need to make sure we have pure json as input
			argBytes, err := json.Marshal(resp.Arguments)
			if err != nil {
				return mlevelerrors.NewMethodLevelErrorServerFail()
			}

			//marshal the result of that particular call
			jsonMessage, mlErr := resolveJSONPointer(argBytes, resultRef.Path)
			if mlErr != nil {
				return mlErr
			}

			if err := json.Unmarshal(jsonMessage, unmarshalAs); err != nil {
				return mlevelerrors.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("resolved %s is of incorrect type", anchorName))
			}
			return nil

		}
	}
	return mlevelerrors.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("no method call id %s found in result", resultRef.ResultOf))

}

func resolveJSONPointer(msg json.RawMessage, pointer string) (json.RawMessage, *mlevelerrors.MethodLevelError) {
	//func resolveJSONPointer(resp map[string]interface{}, pointer string) (json.RawMessage, *mlevelerrors.MethodLevelError) {
	//implements rfc6901

	//the magic needs to happen here

	/*
		valid values for pointer are:
		- /element/subelement
		- /element/arr/0/property1
		- /element/ * /property
	*/

	var resp map[string]interface{}

	if err := json.Unmarshal(msg, &resp); err != nil {
		return nil, mlevelerrors.NewMethodLevelErrorServerFail()
	}

	var result interface{}
	if len(pointer) == 0 {
		result = resp
	} else {
		if !strings.HasPrefix(pointer, "/") {
			return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference("pointer must start with a forward slash ('/')")
		}

		var pathUpTillNow string

		pointerElements := strings.Split(strings.TrimPrefix(pointer, "/"), "/")

		for i, pointerElement := range pointerElements {
			//deal with 2 escapes
			pointerElement = strings.ReplaceAll(pointerElement, "~1", "/")
			pointerElement = strings.ReplaceAll(pointerElement, "~0", "~")

			if i == 0 {
				//we start off with a map[string]interface{}. After that there are different posibilities so we have separate logic for i==0
				pathUpTillNow = "/"
				val, ok := resp[pointerElement]
				if !ok {
					return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("no element with pointer %s found at path %s", pointerElement, pathUpTillNow))
				}
				result = val
				pathUpTillNow = pathUpTillNow + pointerElement
			} else {
				pointerElementInt, err := strconv.Atoi(pointerElement)
				if err == nil {
					//we have a number so we expect an array
					arr, ok := result.([]interface{})
					if !ok {
						return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("cannot use index number on a non array at %s", pathUpTillNow))
					}

					if pointerElementInt > len(arr)-1 {
						//array out of bound
						return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference("array out of bounds")
					}
					result = arr[pointerElementInt]

				} else if pointerElement == "*" {
					//we have special char '*' with it's own logic
					if result == nil {
						//FIXME should we send an empty array
						return json.RawMessage([]byte("null")), nil
					}

					arr, ok := result.([]interface{})
					if !ok {
						return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("%s/* does not reference an array", pathUpTillNow))
					}

					if i != len(pointerElements)-2 {
						//there must only one level remaining
						return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("can only have one extra subelement after using '*'"))
					}

					//get the property that we need
					prop := pointerElements[len(pointerElements)-1]

					var resultArray []interface{}
					for _, arrElement := range arr {
						arrElementMapString, ok := arrElement.(map[string]interface{})
						if !ok {
							//FIXME if it is not map[string]interface{}....
							// 2 options: convert it here, or do it before we call this function: maybe we should have this fn only except json.Rawmessage
							// so we do not care about any 'real' types here. I think that would be the best solution
							mlog.New("mlog-singleton", nil).Debug("unexpected type", slog.Any("arrElement type", fmt.Sprintf("%T", arrElement)))
							return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference("elements in array referenced by '*' must be of type map[string]Object")
						}

						val, ok := arrElementMapString[prop]
						if !ok {
							return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("elements in array referenced by '*' do not have key %s", prop))
						}

						if valArr, ok := val.([]interface{}); ok {
							//the value that is reference by prop is an array it self. We must flattened values in the result
							for _, flattenedArrVal := range valArr {
								resultArray = append(resultArray, flattenedArrVal)
							}
						} else {
							resultArray = append(resultArray, val)
						}
					}
					result = resultArray
					//we are done now so we break the loop
					break
				} else {
					//we dig one level deeper
					mapStringIface, ok := result.(map[string]interface{})
					if !ok {
						return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference("invalid json")
					}

					val, ok := mapStringIface[pointerElement]
					if !ok {
						return nil, mlevelerrors.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("no key %s found at path %s", pointerElement, pathUpTillNow))
					}
					result = val

				}
				pathUpTillNow = pathUpTillNow + "/" + pointerElement
			}
		}
	}

	//marshal the result into a JSON rawmessage
	resultBytes, err := json.Marshal(result)
	if err != nil {
		//should not happen
		return nil, mlevelerrors.NewMethodLevelErrorServerFail()
	}
	return resultBytes, nil
}

// addMethodResponse adds  a invocaction response. It is a builder pattern
func (r *Response) addMethodResponse(i InvocationResponse) {
	r.MethodResponses = append(r.MethodResponses, i)
}

// Reference a result from a previous method call. This in order to save network roundtrips
type ResultReference struct {
	//The method call id (see Section 3.1.1) of a previous method call in the current request.
	ResultOf string `json:"resultOf"`

	//Name is the required name of a response to that method call.
	Name string `json:"name"`

	//A pointer into the arguments of the response selected via the name and resultOf properties.
	//This is a JSON Pointer [@!RFC6901], except it also allows the use of * to map through an array
	Path string `json:"path"`
}

// SessionStater is implement by objects that can return the state of the session object
type SessionStater interface {
	SessionState(ctx context.Context, email string) (string, error)
}

type AccountOpener func(log mlog.Log, name string) (*store.Account, error)

// JAccountFactoryFunc allows injecting a factory for creating a MailboxRepo. This is used for testing
type JAccountFactoryFunc func() (jaccount.JAccounter, string, *mlevelerrors.MethodLevelError)

// APIHandler implements http.Handler
type APIHandler struct {
	Capabilities    capabilitier.Capabilitiers
	SessionStater   SessionStater
	AccountOpener   AccountOpener
	contextUserKey  string
	logger          mlog.Log
	jaccountFactory JAccountFactoryFunc
}

func NewAPIHandler(capabilties capabilitier.Capabilitiers, sessionStater SessionStater, contextUserKey string, accountOpener AccountOpener, logger mlog.Log) *APIHandler {
	result := &APIHandler{
		Capabilities:   capabilties,
		SessionStater:  sessionStater,
		contextUserKey: contextUserKey,
		AccountOpener:  accountOpener,
		logger:         logger,
	}
	//logger.Debug("test", slog.Any("a", (&spew.ConfigState{DisableMethods: true, DisablePointerMethods: true}).Sdump(result)))
	return result
}

func (ah *APIHandler) WithOverrideJAccountFactory(f JAccountFactoryFunc) *APIHandler {
	ah.jaccountFactory = f
	return ah
}

// ServeHTTP implements http.Handler
func (ah APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//populate the reponse with the CORS headers
	AddCORSAllowedOriginHeader(w, r)

	coreSettings := ah.Capabilities.CoreSettings()
	if coreSettings == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if r.ContentLength > int64(coreSettings.MaxSizeRequest) {
		//../../rfc/8620:1099
		writeOutput(http.StatusBadRequest, NewRequestLevelErrorCapabilityLimit(LimitTypeMaxSizeRequest, fmt.Sprintf("max request size is %d bytes", coreSettings.MaxSizeRequest)), w, ah.logger)
		return
	}

	reqBody, err := httputil.DumpRequest(r, true)
	if err == nil {
		ah.logger.Debug("dump http request", slog.Any("payload", string(reqBody)))
	}

	if !isContentTypeJSON(r.Header.Get(HeaderContentType)) {
		writeOutput(http.StatusBadRequest, NewRequestLevelErrorNotJSONContentType(), w, ah.logger)
		return
	}

	var request Request

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		switch e := err.(type) {
		case *json.InvalidUnmarshalError:
			//InvalidUnmarshalError is only returned when a non pointer is provided to Decode()
			writeOutput(http.StatusInternalServerError, nil, w, ah.logger)
			return
		case *json.SyntaxError:
			//../../rfc/8620:1091
			//SyntaxError means the JSON is invalid
			writeOutput(http.StatusBadRequest, NewRequestLevelErrorNotJSON(err.Error()), w, ah.logger)
			return
		case *json.UnmarshalTypeError:
			//../../rfc/8620:1095
			//SyntaxError means the JSON is invalid
			writeOutput(http.StatusBadRequest, NewRequestLevelErrorNotRequest(fmt.Sprintf("error in %s", e.Field)), w, ah.logger)
			return
		default:
			//have a catch all for other errors that unmarschal may throw
			writeOutput(http.StatusInternalServerError, nil, w, ah.logger)
			return
		}
	}

	if len(request.Using) == 0 || len(request.MethodCalls) == 0 {
		writeOutput(http.StatusBadRequest, NewRequestLevelErrorNotRequest("'using' empty or no method calls"), w, ah.logger)
		return
	}

	//check if 'using' field of the request
loopUsing:
	for _, capabilityURN := range request.Using {
		for _, capability := range ah.Capabilities {
			if capability.Urn() == capabilityURN {
				continue loopUsing
			}
		}
		//../../rfc/8620:1087
		writeOutput(http.StatusBadRequest, NewRequestLevelErrorUnknownCapability(fmt.Sprintf("%s is not a known capability", capabilityURN)), w, ah.logger)
		return
	}

	//defaultJAccountFactory instantiates a JAccount
	defaultJAccountFactory := func() (*jaccount.JAccount, string, *mlevelerrors.MethodLevelError) {
		//pass in the jaccount
		userIface := r.Context().Value(ah.contextUserKey)
		if userIface == nil {
			ah.logger.Debug("no user found in context")
			return nil, "", mlevelerrors.NewMethodLevelErrorAccountForFound()
		}

		userObj, ok := userIface.(user.User)
		if !ok {
			ah.logger.Debug("user is not of type user.User", slog.Any("unexpectedtype", fmt.Sprintf("%T", userIface)))
			return nil, "", mlevelerrors.NewMethodLevelErrorAccountForFound()
		}

		mAccount, err := ah.AccountOpener(ah.logger, userObj.Name)
		if err != nil {
			ah.logger.Debug("error opening account", slog.Any("err", err.Error()), slog.Any("accountname", userObj.Email))
			return nil, "", mlevelerrors.NewMethodLevelErrorAccountForFound()
		}

		mailboxRepo := bstore.QueryDB[store.Mailbox](r.Context(), mAccount.DB)
		return jaccount.NewJAccount(mAccount, mailboxRepo, ah.logger), userObj.Email, nil
	}

	var (
		jAccount       jaccount.JAccounter
		email          string
		accountOpenErr *mlevelerrors.MethodLevelError
	)

	//the echo method does not require this but instantiating this here makes closing the account way simpeler. Otherwise repititive blocks were needed
	if ah.jaccountFactory == nil {
		//bespoke factory is used for testing
		jAccount, email, accountOpenErr = defaultJAccountFactory()
		if accountOpenErr == nil {
			defer jAccount.Close()
		}
	} else {
		jAccount, email, accountOpenErr = ah.jaccountFactory()
	}

	response := new(Response)

	//all request level checks are done now so start with the processing of the invocations
	for _, invocation := range request.MethodCalls {

		var invocationResponse InvocationResponse = newInvocationResponse(invocation.MethodCallID)

		//TODO there are more methods than these. Maybe methods should be registered?
		methodCallRegexp := regexp.MustCompile("^[a-zA-Z]+/(echo|get|changes|set|copy|query|queryChanges)$")

		if !methodCallRegexp.MatchString(invocation.Name) {
			response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
			continue
		}

		if accountOpenErr != nil {
			//if there is an issue with opening the account, all methods return the same error
			response.addMethodResponse(invocationResponse.withArgError(accountOpenErr))
			continue
		}

		nameParts := strings.Split(invocation.Name, "/")
		if len(nameParts) != 2 {
			response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
			continue
		}

		dt := ah.Capabilities.GetDatatypeByName(nameParts[0])
		if dt == nil {
			response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
			continue
		}

		ah.logger.Debug("method called", slog.Any("method", invocation.Name))

		switch nameParts[1] {
		case "echo":
			echoEr, ok := dt.(datatyper.Echoer)
			if !ok {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			resp, mErr := echoEr.Echo(r.Context(), invocation.Arguments)
			if mErr != nil {
				response.addMethodResponse(invocationResponse.withArgError(mErr))
			} else {
				response.addMethodResponse(invocationResponse.withArgOK(invocation.Name, resp))
			}

		case "get":
			dtGetter, ok := dt.(datatyper.Getter)
			if !ok {
				//datatype does not have this method
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type getRequest struct {
				AccountId  basetypes.Id   `json:"accountId"`
				Ids        []basetypes.Id `json:"ids"`
				Properties []string       `json:"properties"`

				//FIXME the '#' fields should be determined dynamically however I am not 100% sure that should be the case
				AccountIdResultRef  *ResultReference `json:"#accountId,omitempty"`
				IdsResultRef        *ResultReference `json:"#ids,omitempty"`
				PropertiesResultRef *ResultReference `json:"#properties,omitempty"`
			}

			requestArgs := new(getRequest)

			if err := json.Unmarshal(invocation.Arguments, requestArgs); err != nil {
				if mle, ok := err.(*mlevelerrors.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				if typeError, ok := err.(*json.UnmarshalTypeError); ok {
					//this is needed to catch unmarshal type errors in accountId
					response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("incorrect type for field %s", typeError.Field))))
					continue
				}
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
				continue
			}

			if !requestArgs.AccountId.IsEmpty() && requestArgs.AccountIdResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments("cannot use 'accountId' and '#accountId' together")))
				continue
			}
			if len(requestArgs.Ids) > 0 && requestArgs.IdsResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments("cannot use 'ids' and '#ids' together")))
				continue
			}
			if len(requestArgs.Properties) > 0 && requestArgs.PropertiesResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments("cannot use 'properties' and '#properties' together")))
				continue
			}

			finalAccountId := requestArgs.AccountId
			finalIds := requestArgs.Ids
			finalProperties := requestArgs.Properties

			if requestArgs.AccountIdResultRef != nil {
				var accId basetypes.Id
				mlErr := response.getResultByRef(ah.logger, requestArgs.AccountIdResultRef, "#accountId", &accId)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalAccountId = accId
			}

			if requestArgs.IdsResultRef != nil {
				//so we now have the thing that we need to insert
				var ids []basetypes.Id
				mlErr := response.getResultByRef(ah.logger, requestArgs.IdsResultRef, "#ids", &ids)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalIds = ids
			}

			if requestArgs.PropertiesResultRef != nil {
				var props []string
				mlErr := response.getResultByRef(ah.logger, requestArgs.PropertiesResultRef, "#properties", &props)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalProperties = props
			}

			if finalAccountId.IsEmpty() {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments("accountId cannot be empty")))
				continue
			}

			if len(finalIds) > int(coreSettings.MaxObjectsInGet) {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorRequestTooLarge()))
				continue
			}

			//unmarshal the bespoke parts
			bespokeParams := dtGetter.CustomGetRequestParams()
			if bespokeParams != nil {
				if err := json.Unmarshal(invocation.Arguments, bespokeParams); err != nil {
					//FIXME I am repeating this block a lot
					if mle, ok := err.(*mlevelerrors.MethodLevelError); ok {
						response.addMethodResponse(invocationResponse.withArgError(mle))
						continue
					}
					if typeError, ok := err.(*json.UnmarshalTypeError); ok {
						//this is needed to catch unmarshal type errors in accountId
						response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("incorrect type for field %s", typeError.Field))))
						continue
					}
					response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
					continue
				}
			}

			retAccountId, state, list, notFound, mErr := dtGetter.Get(r.Context(), jAccount, finalAccountId, dedupIDSlice(finalIds), finalProperties, bespokeParams)
			if mErr != nil {
				response.addMethodResponse(invocationResponse.withArgError(mErr))
				continue
			}

			//FIXME not sure if this is the place
			//do property filtering
			//id should be always returned even if it is not requested
			//AAA
			// ../../rfc/8620:1608
			propertyFilteredList, err := filterProperties(list, append(finalProperties), []string{"id"})
			if err != nil {
				ah.logger.Error("applying filtering failed ", slog.Any("err", err.Error()))
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
				continue
			}

			response.addMethodResponse(invocationResponse.withArgOK(invocation.Name, map[string]interface{}{
				//FIXME maybe this should be set entirely by the object that returns the result because some fields are not so fixed as expected
				"accountId": retAccountId,
				"state":     state,
				"list":      propertyFilteredList,
				"notFound":  notFound,
			}))

		case "changes":
			dtChanges, ok := dt.(datatyper.Changeser)
			if !ok {
				//datatype does not have this method
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type changesRequest struct {
				AccountId  basetypes.Id    `json:"accountId"`
				SinceState string          `json:"sinceState"`
				MaxChanges *basetypes.Uint `json:"maxChanges"`

				AccountIdResultRef  *ResultReference `json:"#accountId"`
				SinceStateResultRef *ResultReference `json:"#sinceState"`
				MaxChangesResultRef *ResultReference `json:"#maxChanges"`
			}

			var requestArgs changesRequest

			if err := json.Unmarshal(invocation.Arguments, &requestArgs); err != nil {
				if mle, ok := err.(*mlevelerrors.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				if typeError, ok := err.(*json.UnmarshalTypeError); ok {
					//this is needed to send correct unmarshal type errors in accountId
					response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("incorrect type for field %s", typeError.Field))))
					continue
				}
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
				continue
			}

			if !requestArgs.AccountId.IsEmpty() && requestArgs.AccountIdResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments("cannot use 'accountId' and '#accountId' together")))
				continue
			}
			if requestArgs.SinceState != "" && requestArgs.SinceStateResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments("cannot use 'sinceState' and '#sinceState' together")))
				continue
			}
			if requestArgs.MaxChanges != nil && requestArgs.MaxChangesResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments("cannot use 'maxChanges' and '#maxChanges' together")))
				continue
			}

			finalAccountId := requestArgs.AccountId
			finalSinceState := requestArgs.SinceState
			finalMaxChanges := requestArgs.MaxChanges

			if requestArgs.AccountIdResultRef != nil {
				var accId basetypes.Id
				mlErr := response.getResultByRef(ah.logger, requestArgs.AccountIdResultRef, "#accountId", &accId)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalAccountId = accId
			}

			if requestArgs.SinceStateResultRef != nil {
				//so we now have the thing that we need to insert
				var sinceState string
				mlErr := response.getResultByRef(ah.logger, requestArgs.SinceStateResultRef, "#sinceState", &sinceState)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalSinceState = sinceState
			}

			if requestArgs.MaxChangesResultRef != nil {
				var maxChanges *basetypes.Uint
				mlErr := response.getResultByRef(ah.logger, requestArgs.MaxChangesResultRef, "#maxChanges", &maxChanges)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalMaxChanges = maxChanges
			}

			retAccountId, oldState, newState, hasMoreChanges, created, updated, destroyed, mErr := dtChanges.Changes(r.Context(), jAccount, finalAccountId, finalSinceState, finalMaxChanges)
			if mErr != nil {
				response.addMethodResponse(invocationResponse.withArgError(mErr))
				continue
			}

			response.addMethodResponse(invocationResponse.withArgOK(invocation.Name, map[string]interface{}{
				"accountId":      retAccountId,
				"oldState":       oldState,
				"newState":       newState,
				"hasMoreChanges": hasMoreChanges,
				"created":        created,
				"updated":        updated,
				"destroyed":      destroyed,
			}))

		case "set":
			dtSet, ok := dt.(datatyper.Setter)
			if !ok {
				//datatype does not have this method
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type setRequest struct {
				AccountId basetypes.Id                           `json:"accountId"`
				IfInState *string                                `json:"ifInState"`
				Create    map[basetypes.Id]interface{}           `json:"create"`
				Update    map[basetypes.Id]basetypes.PatchObject `json:"update"`
				Destroy   []basetypes.Id                         `json:"destroy"`

				AccountIdResultRef *ResultReference `json:"#accountId"`
				IfInStateResultRef *ResultReference `json:"#ifInState"`
				CreateResultRef    *ResultReference `json:"#create"`
				UpdateResultRef    *ResultReference `json:"#update"`
				DestroyResultRef   *ResultReference `json:"#destroy"`
			}

			var requestArgs setRequest

			if err := json.Unmarshal(invocation.Arguments, &requestArgs); err != nil {
				if mle, ok := err.(*mlevelerrors.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				ah.logger.Error("unmarshal error", slog.Any("method", "set"), slog.Any("err", err))

				//FIXME handle datatype conversion errors properly
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
				continue
			}

			if len(requestArgs.Create)+len(requestArgs.Update)+len(requestArgs.Destroy) > int(coreSettings.MaxObjectsInSet) {
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorRequestTooLarge()))
				continue
			}

			retAccountId, oldState, newState, created, updated, destroyed, notCreated, notUpdated, notDestroyed, mErr := dtSet.Set(r.Context(), jAccount, requestArgs.AccountId, requestArgs.IfInState, requestArgs.Create, requestArgs.Update, requestArgs.Destroy)
			if mErr != nil {
				response.addMethodResponse(invocationResponse.withArgError(mErr))
				continue
			}

			response.addMethodResponse(invocationResponse.withArgOK(invocation.Name, map[string]interface{}{
				"accountId":    retAccountId,
				"oldState":     oldState,
				"newState":     newState,
				"created":      created,
				"updated":      updated,
				"destroyed":    destroyed,
				"notCreated":   notCreated,
				"notUpdated":   notUpdated,
				"notDestroyed": notDestroyed,
			}))

		case "copy":
			dtCopy, ok := dt.(datatyper.Copier)
			if !ok {
				//datatype does not have this method
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type copyRequest struct {
				FromAccountId           basetypes.Id                 `json:"fromAccountId"`
				IfFromState             *string                      `json:"ifFromState"`
				AccountId               basetypes.Id                 `json:"accountId"`
				IfInState               *string                      `json:"ifInState"`
				Create                  map[basetypes.Id]interface{} `json:"create"`
				OnSuccesDestroyOriginal bool                         `json:"onSuccesDestroyOriginal"`
				DestroyFromIfInState    *string                      `json:"destroyFromIfInState"`

				FromAccountIdResultRef           *ResultReference `json:"#fromAccountId"`
				IfFromStateResultRef             *ResultReference `json:"#ifFromState"`
				AccountIdResultRef               *ResultReference `json:"#accountId"`
				IfInStateResultRef               *ResultReference `json:"#ifInState"`
				CreateResultRef                  *ResultReference `json:"#create"`
				OnSuccesDestroyOriginalResultRef *ResultReference `json:"#onSuccesDestroyOriginal"`
				DestroyFromIfInStateResultRef    *ResultReference `json:"#destroyFromIfInState"`
			}

			var requestArgs copyRequest

			if err := json.Unmarshal(invocation.Arguments, &requestArgs); err != nil {
				if mle, ok := err.(*mlevelerrors.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				//FIXME handle datatype conversion errors properly
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
				continue
			}
			retFromAccountId, retAccountId, oldState, newState, created, notCreated, mErr := dtCopy.Copy(r.Context(), requestArgs.FromAccountId, requestArgs.IfFromState, requestArgs.AccountId, requestArgs.IfInState, requestArgs.Create, requestArgs.OnSuccesDestroyOriginal, requestArgs.DestroyFromIfInState)

			if mErr != nil {
				response.addMethodResponse(invocationResponse.withArgError(mErr))
				continue
			}

			response.addMethodResponse(invocationResponse.withArgOK(invocation.Name, map[string]interface{}{
				"fromAccountId": retFromAccountId,
				"accountId":     retAccountId,
				"oldState":      oldState,
				"newState":      newState,
				"created":       created,
				"notCreated":    notCreated,
			}))

		case "query":

			dtQuery, ok := dt.(datatyper.Querier)
			if !ok {
				//datatype does not have this method
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type queryRequest struct {
				AccountId            basetypes.Id           `json:"accountId"`
				Filter               *basetypes.Filter      `json:"filter"`
				Sort                 []basetypes.Comparator `json:"sort"`
				Position             basetypes.Int          `json:"position"`
				Anchor               *basetypes.Id          `json:"anchor"`
				AnchorOffset         basetypes.Int          `json:"anchorOffset"`
				Limit                *basetypes.Uint        `json:"limit"`
				CalculateTotal       bool                   `json:"calculateTotal"`
				DestroyFromIfInState *string                `json:"destroyFromIfInState"`

				AccountIdResultRef            *ResultReference `json:"#accountId"`
				FilterResultRef               *ResultReference `json:"#filter"`
				SortResultRef                 *ResultReference `json:"#sort"`
				PositionResultRef             *ResultReference `json:"#position"`
				AnchorResultRef               *ResultReference `json:"#anchor"`
				AnchorOffsetResultRef         *ResultReference `json:"#anchorOffset"`
				LimitResultRef                *ResultReference `json:"#limit"`
				CalculateTotalResultRef       *ResultReference `json:"#calculateTotal"`
				DestroyFromIfInStateResultRef *ResultReference `json:"#destroyFromIfInState"`
			}

			var requestArgs queryRequest

			if err := json.Unmarshal(invocation.Arguments, &requestArgs); err != nil {
				if mle, ok := err.(*mlevelerrors.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				//FIXME handle datatype conversion errors properly
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
				continue
			}

			//unmarshal the bespoke parts
			bespokeParams := dtQuery.CustomQueryRequestParams()
			if bespokeParams != nil {
				if err := json.Unmarshal(invocation.Arguments, bespokeParams); err != nil {
					//FIXME I am repeating this block a lot
					if mle, ok := err.(*mlevelerrors.MethodLevelError); ok {
						response.addMethodResponse(invocationResponse.withArgError(mle))
						continue
					}
					if typeError, ok := err.(*json.UnmarshalTypeError); ok {
						//this is needed to catch unmarshal type errors in accountId
						response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("incorrect type for field %s", typeError.Field))))
						continue
					}
					response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
					continue
				}
			}

			retAccountId, queryState, canCalculateChanges, retPosition, ids, total, retLimit, mErr := dtQuery.Query(r.Context(), jAccount, requestArgs.AccountId, requestArgs.Filter, requestArgs.Sort, requestArgs.Position, requestArgs.Anchor, requestArgs.AnchorOffset, requestArgs.Limit, requestArgs.CalculateTotal, bespokeParams)
			if mErr != nil {
				response.addMethodResponse(invocationResponse.withArgError(mErr))
				continue
			}

			response.addMethodResponse(invocationResponse.withArgOK(invocation.Name, map[string]interface{}{
				"accountId":           retAccountId,
				"queryState":          queryState,
				"canCalculateChanges": canCalculateChanges,
				"position":            retPosition,
				"ids":                 ids,
				"total":               total,
				"limit":               retLimit,
			}))

			ah.logger.Debug("query results",
				slog.Any("queryState", queryState),
				slog.Any("canCalculateChanges", canCalculateChanges),
				slog.Any("position", retPosition),
				slog.Any("ids", func(ids []basetypes.Id) string {
					var result string
					for i, id := range ids {
						if i == 0 {
							result = string(id)
						} else {
							result = result + "," + string(id)
						}
					}
					return result
				}(ids)),
				slog.Any("total", total),
				slog.Any("limit", retLimit),
			)

		case "queryChanges":

			dtQueryChanges, ok := dt.(datatyper.QueryChangeser)
			if !ok {
				//datatype does not have this method
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type queryChangesRequest struct {
				AccountId       basetypes.Id           `json:"accountId"`
				Filter          *basetypes.Filter      `json:"filter"`
				Sort            []basetypes.Comparator `json:"sort"`
				SinceQueryState string                 `json:"sinceQueryState"`
				MaxChanges      *basetypes.Uint        `json:"maxChanges"`
				UpToId          *basetypes.Id          `json:"upToId"`
				CalculateTotal  bool                   `json:"calculateTotal"`

				AccountIdResultRef       *ResultReference `json:"#accountId"`
				FilterResultRef          *ResultReference `json:"#filter"`
				SortResultRef            *ResultReference `json:"#sort"`
				SinceQueryStateResultRef *ResultReference `json:"#sinceQueryState"`
				MaxChangesResultRef      *ResultReference `json:"#maxChanges"`
				UpToIdResultRef          *ResultReference `json:"#upToId"`
				CalculateTotalResultRef  *ResultReference `json:"#calculateTotal"`
			}

			var requestArgs queryChangesRequest

			if err := json.Unmarshal(invocation.Arguments, &requestArgs); err != nil {
				if mle, ok := err.(*mlevelerrors.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				//FIXME handle datatype conversion errors properly
				response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
				continue
			}
			retAccountId, oldQueryState, newQueryState, total, removed, added, mErr := dtQueryChanges.QueryChanges(r.Context(), requestArgs.AccountId, requestArgs.Filter, requestArgs.Sort, requestArgs.SinceQueryState, requestArgs.MaxChanges, requestArgs.UpToId, requestArgs.CalculateTotal)

			if mErr != nil {
				response.addMethodResponse(invocationResponse.withArgError(mErr))
				continue
			}

			response.addMethodResponse(invocationResponse.withArgOK(invocation.Name, map[string]interface{}{
				"accountId":     retAccountId,
				"oldQueryState": oldQueryState,
				"newQueryState": newQueryState,
				"total":         total,
				"removed":       removed,
				"added":         added,
			}))

		default:
			//should not get here ever
			response.addMethodResponse(invocationResponse.withArgError(mlevelerrors.NewMethodLevelErrorServerFail()))
		}
	}

	//FIXME need to check if I need a separate var for email or that i can use the name property of Account
	if sessionState, err := ah.SessionStater.SessionState(r.Context(), email); err == nil {
		response.SessionState = sessionState
	} else {
		ah.logger.Error("error getting state", slog.Any("err", err))
		response.SessionState = "session_err_state"
	}
	writeOutput(200, response, w, ah.logger)
}

// writeOutput encodes a the body into json and writes the output the the reponse writer
func writeOutput(statusCode int, body interface{}, w http.ResponseWriter, logger mlog.Log) {

	if statusCode == http.StatusInternalServerError {
		w.WriteHeader(statusCode)
		return
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		//we cannot do the json encoding
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Debug("http response", slog.Any("response", string(jsonBytes)))

	w.Header().Add(HeaderContentType, HeaderContentTypeJSON)
	w.WriteHeader(statusCode)
	w.Write(jsonBytes)
}

func isContentTypeJSON(ct string) bool {
	if strings.ToLower(ct) == strings.ToLower(HeaderContentTypeJSON) || strings.ToLower(ct) == strings.ToLower(HeaderContentTypeJSONUTF8) {
		return true
	}
	return false
}

// filterProperties removes any top level elements from list that are not in properties except for elements in alwaysInclude
func filterProperties(list []any, properties []string, alwaysInclude []string) ([]any, error) {
	if len(properties) == 0 {
		//nothing to do
		return list, nil
	}

	//Marshal first
	listBytes, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}

	var myMaps []map[string]interface{}

	if err := json.Unmarshal(listBytes, &myMaps); err != nil {
		return nil, err
	}

	var result []any

	for _, myMap := range myMaps {
		for k := range myMap {
			var propNeedsToBeIncluded bool
			for _, prop := range properties {
				if prop == k {
					propNeedsToBeIncluded = true
					break
				}
			}
			if !propNeedsToBeIncluded {
				for _, a := range alwaysInclude {
					if a == k {
						propNeedsToBeIncluded = true
						break
					}
				}
			}
			if !propNeedsToBeIncluded {
				delete(myMap, k)
			}
		}
		result = append(result, myMap)

	}
	return result, nil
}

// dedupIDSlice deduplicates a basetype.Id slice
func dedupIDSlice(in []basetypes.Id) []basetypes.Id {
	var result []basetypes.Id

	helper := make(map[basetypes.Id]interface{})

	for _, id := range in {
		helper[id] = nil
	}

	for id := range helper {
		result = append(result, id)
	}

	return result
}
