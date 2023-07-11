package httphandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/core"
	"github.com/mjl-/mox/jmapserver/datatyper"
)

const (
	HeaderContentType     = "content-type"
	HeaderContentTypeJSON = "application/json"
)

// Request is the top level request object for the api handler
type Request struct {
	//Using contains the set of capabilities the client wishes to use
	Using []string `json:"using"`

	//MethodCalls is an array of method calls to process on the server
	MethodCalls []InvocationRequest `json:"methodCalls"`

	//CreatedIds is an  (optional) map of a (client-specified) creation id to the id the server assigned when a record was successfully created.
	CreatedIds map[datatyper.Id]datatyper.Id `json:"createdIds"`
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

// withArgError adds an error to a invocation reponse
func (inv InvocationResponse) withArgError(mErr *datatyper.MethodLevelError) InvocationResponse {
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

// Response is the top level reponse that is sent by the API handler
type Response struct {
	MethodResponses []InvocationResponse `json:"methodResponses"`
	CreatedIds      []datatyper.Id       `json:"createdIds,omitempty"`
	SessionState    string               `json:"sessionState"`
}

// getResultByRef resolves the ResultReference
func (r Response) getResultByRef(resultRef *ResultReference, anchorName string, unmarshalAs any) *datatyper.MethodLevelError {
	for _, resp := range r.MethodResponses {
		if resp.MethodCallID == resultRef.ResultOf {
			//need to check if the name of the method matches
			if resp.Name != resultRef.Name {
				return datatyper.NewMethodLevelErrorInvalidResultReference("method name is not matching with method call id")
			}
			//marshal the result of that particular call
			jsonMessage, mlErr := resolveJSONPointer(resp.Arguments, resultRef.Path)
			if mlErr != nil {
				return mlErr
			}

			if err := json.Unmarshal(jsonMessage, unmarshalAs); err != nil {
				return datatyper.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("resolved %s is of incorrect type", anchorName))
			}
			return nil

		}
	}
	return datatyper.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("no method call id %s found in result", resultRef.ResultOf))

}

func resolveJSONPointer(resp map[string]interface{}, pointer string) (json.RawMessage, *datatyper.MethodLevelError) {
	//implements rfc6901

	//the magic needs to happen here

	/*
		valid values for pointer are:
		- /element/subelement
		- /element/arr/0/property1
		- /element/ * /property
	*/

	var result interface{}
	if len(pointer) == 0 {
		result = resp
	} else {
		if !strings.HasPrefix(pointer, "/") {
			return nil, datatyper.NewMethodLevelErrorInvalidResultReference("pointer must start with a forward slash ('/')")
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
					return nil, datatyper.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("no element with pointer %s found at path %s", pointerElement, pathUpTillNow))
				}
				result = val
				pathUpTillNow = pathUpTillNow + pointerElement
			} else {
				pointerElementInt, err := strconv.Atoi(pointerElement)
				if err == nil {
					//we have a number so we expect an array
					arr, ok := result.([]interface{})
					if !ok {
						return nil, datatyper.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("cannot use index number on a non array at %s", pathUpTillNow))
					}

					if pointerElementInt > len(arr)-1 {
						//array out of bound
						return nil, datatyper.NewMethodLevelErrorInvalidResultReference("array out of bounds")
					}
					result = arr[pointerElementInt]

				} else if pointerElement == "*" {
					//we have special char '*' with it's own logic
					arr, ok := result.([]interface{})
					if !ok {
						return nil, datatyper.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("%s/* does not reference an array", pathUpTillNow))
					}

					if i != len(pointerElements)-2 {
						//there must only one level remaining
						return nil, datatyper.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("can only have one extra subelement after using '*'"))
					}

					//get the property that we need
					prop := pointerElements[len(pointerElements)-1]

					var resultArray []interface{}
					for _, arrElement := range arr {
						arrElementMapString, ok := arrElement.(map[string]interface{})
						if !ok {
							return nil, datatyper.NewMethodLevelErrorInvalidResultReference("elements in array referenced by '*' must be of type map[string]Object")
						}

						val, ok := arrElementMapString[prop]
						if !ok {
							return nil, datatyper.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("elements in array referenced by '*' do not have key %s", prop))
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
						return nil, datatyper.NewMethodLevelErrorInvalidResultReference("invalid json")
					}

					val, ok := mapStringIface[pointerElement]
					if !ok {
						return nil, datatyper.NewMethodLevelErrorInvalidResultReference(fmt.Sprintf("no key %s found at path %s", pointerElement, pathUpTillNow))
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
		return nil, datatyper.NewMethodLevelErrorServerFail()
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

// FIXME this needs an implentation
type SessionStater interface {
	SessionState() string
}

// APIHandler implements http.Handler
type APIHandler struct {
	Capabilities           capabilitier.Capabilitiers
	CoreCapabilitySettings core.CoreCapabilitySettings
	SessionStater          SessionStater
}

func NewAPIHandler(capabilties capabilitier.Capabilitiers, coreSettings core.CoreCapabilitySettings, sessionStater SessionStater) *APIHandler {
	return &APIHandler{
		Capabilities:           capabilties,
		CoreCapabilitySettings: coreSettings,
		SessionStater:          sessionStater,
	}
}

// ServeHTTP implements http.Handler
func (ah APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get(HeaderContentType) != HeaderContentTypeJSON {
		writeOutput(http.StatusBadRequest, NewRequestLevelErrorNotJSONContentType(), w)
		return
	}

	if r.ContentLength > int64(ah.CoreCapabilitySettings.MaxSizeRequest) {
		writeOutput(http.StatusBadRequest, NewRequestLevelErrorCapabilityLimit(LimitTypeMaxSizeRequest, fmt.Sprintf("max request size is %d bytes", ah.CoreCapabilitySettings.MaxSizeRequest)), w)
		return
	}

	var request Request

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		switch e := err.(type) {
		case *json.InvalidUnmarshalError:
			//InvalidUnmarshalError is only returned when a non pointer is provided to Decode()
			writeOutput(http.StatusInternalServerError, nil, w)
			return
		case *json.SyntaxError:
			//SyntaxError means the JSON is invalid
			writeOutput(http.StatusBadRequest, NewRequestLevelErrorNotJSON(err.Error()), w)
			return
		case *json.UnmarshalTypeError:
			writeOutput(http.StatusBadRequest, NewRequestLevelErrorNotRequest(fmt.Sprintf("error in %s", e.Field)), w)
			return
		default:
			//have a catch all for other errors that unmarschal may throw
			writeOutput(http.StatusInternalServerError, nil, w)
			return
		}
	}

	if len(request.Using) == 0 || len(request.MethodCalls) == 0 {
		writeOutput(http.StatusBadRequest, NewRequestLevelErrorNotRequest("'using' empty or no method calls"), w)
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
		writeOutput(http.StatusBadRequest, NewRequestLevelErrorUnknownCapability(fmt.Sprintf("%s is not a known capability", capabilityURN)), w)
		return
	}

	response := new(Response)

	//all request level checks are done now so start with the processing of the invocations
	for _, invocation := range request.MethodCalls {

		var invocationResponse InvocationResponse = newInvocationResponse(invocation.MethodCallID)

		methodCallRegexp := regexp.MustCompile("^[a-zA-Z]+/(echo|get|changes|set|copy|query|queryChanges)$")

		if !methodCallRegexp.MatchString(invocation.Name) {
			response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
			continue
		}

		nameParts := strings.Split(invocation.Name, "/")
		if len(nameParts) != 2 {
			response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
			continue
		}

		dt := ah.Capabilities.GetDatatypeByName(nameParts[0])
		if dt == nil {
			response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
			continue
		}

		switch nameParts[1] {
		case "echo":
			echoEr, ok := dt.(datatyper.Echoer)
			if !ok {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
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
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type getRequest struct {
				AccountId  datatyper.Id   `json:"accountId"`
				Ids        []datatyper.Id `json:"ids"`
				Properties []string       `json:"properties"`

				AdditionalFields map[string]json.RawMessage

				//FIXME the '#' fields should be determined dynamically however I am not 100% sure that should be the case
				AccountIdResultRef  *ResultReference `json:"#accountId,omitempty"`
				IdsResultRef        *ResultReference `json:"#ids,omitempty"`
				PropertiesResultRef *ResultReference `json:"#properties,omitempty"`
			}

			requestArgs := new(getRequest)

			if err := json.Unmarshal(invocation.Arguments, requestArgs); err != nil {
				if mle, ok := err.(*datatyper.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				if typeError, ok := err.(*json.UnmarshalTypeError); ok {
					//this is needed to catch unmarshal type errors in accountId
					response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("incorrect type for field %s", typeError.Field))))
					continue
				}
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorServerFail()))
				continue
			}

			if !requestArgs.AccountId.IsEmpty() && requestArgs.AccountIdResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments("cannot use 'accountId' and '#accountId' together")))
				continue
			}
			if len(requestArgs.Ids) > 0 && requestArgs.IdsResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments("cannot use 'ids' and '#ids' together")))
				continue
			}
			if len(requestArgs.Properties) > 0 && requestArgs.PropertiesResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments("cannot use 'properties' and '#properties' together")))
				continue
			}

			finalAccountId := requestArgs.AccountId
			finalIds := requestArgs.Ids
			finalProperties := requestArgs.Properties

			if requestArgs.AccountIdResultRef != nil {
				var accId datatyper.Id
				mlErr := response.getResultByRef(requestArgs.AccountIdResultRef, "#accountId", &accId)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalAccountId = accId
			}

			if requestArgs.IdsResultRef != nil {
				//so we now have the thing that we need to insert
				var ids []datatyper.Id
				mlErr := response.getResultByRef(requestArgs.AccountIdResultRef, "#ids", &ids)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalIds = ids
			}

			if requestArgs.PropertiesResultRef != nil {
				var props []string
				mlErr := response.getResultByRef(requestArgs.AccountIdResultRef, "#properties", &props)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalProperties = props
			}

			if finalAccountId.IsEmpty() {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments("accountId cannot be empty")))
				continue
			}

			if len(finalIds) > int(ah.CoreCapabilitySettings.MaxObjectsInGet) {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorRequestTooLarge()))
				continue
			}

			retAccountId, state, list, notFound, mErr := dtGetter.Get(r.Context(), finalAccountId, finalIds, finalProperties)
			if mErr != nil {
				response.addMethodResponse(invocationResponse.withArgError(mErr))
				continue
			}

			response.addMethodResponse(invocationResponse.withArgOK(invocation.Name, map[string]interface{}{
				//FIXME maybe this should be set entirely by the object that returns the result because some fields are not so fixed as expected
				"accountId": retAccountId,
				"state":     state,
				"list":      list,
				"notFound":  notFound,
			}))

		case "changes":
			dtChanges, ok := dt.(datatyper.Changeser)
			if !ok {
				//datatype does not have this method
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type changesRequest struct {
				AccountId  datatyper.Id    `json:"accountId"`
				SinceState string          `json:"sinceState"`
				MaxChanges *datatyper.Uint `json:"maxChanges"`

				AccountIdResultRef  *ResultReference `json:"#accountId"`
				SinceStateResultRef *ResultReference `json:"#sinceState"`
				MaxChangesResultRef *ResultReference `json:"#maxChanges"`
			}

			var requestArgs changesRequest

			if err := json.Unmarshal(invocation.Arguments, &requestArgs); err != nil {
				if mle, ok := err.(*datatyper.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				if typeError, ok := err.(*json.UnmarshalTypeError); ok {
					//this is needed to send correct unmarshal type errors in accountId
					response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("incorrect type for field %s", typeError.Field))))
					continue
				}
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorServerFail()))
				continue
			}

			if !requestArgs.AccountId.IsEmpty() && requestArgs.AccountIdResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments("cannot use 'accountId' and '#accountId' together")))
				continue
			}
			if requestArgs.SinceState != "" && requestArgs.SinceStateResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments("cannot use 'sinceState' and '#sinceState' together")))
				continue
			}
			if requestArgs.MaxChanges != nil && requestArgs.MaxChangesResultRef != nil {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorInvalidArguments("cannot use 'maxChanges' and '#maxChanges' together")))
				continue
			}

			finalAccountId := requestArgs.AccountId
			finalSinceState := requestArgs.SinceState
			finalMaxChanges := requestArgs.MaxChanges

			if requestArgs.AccountIdResultRef != nil {
				var accId datatyper.Id
				mlErr := response.getResultByRef(requestArgs.AccountIdResultRef, "#accountId", &accId)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalAccountId = accId
			}

			if requestArgs.SinceStateResultRef != nil {
				//so we now have the thing that we need to insert
				var sinceState string
				mlErr := response.getResultByRef(requestArgs.AccountIdResultRef, "#sinceState", &sinceState)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalSinceState = sinceState
			}

			if requestArgs.MaxChangesResultRef != nil {
				var maxChanges *datatyper.Uint
				mlErr := response.getResultByRef(requestArgs.AccountIdResultRef, "#maxChanges", &maxChanges)
				if mlErr != nil {
					response.addMethodResponse(invocationResponse.withArgError(mlErr))
					continue
				}
				finalMaxChanges = maxChanges
			}

			retAccountId, oldState, newState, hasMoreChanges, created, updated, destroyed, mErr := dtChanges.Changes(r.Context(), finalAccountId, finalSinceState, finalMaxChanges)
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
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type setRequest struct {
				AccountId datatyper.Id                             `json:"accountId"`
				IfInState *string                                  `json:"ifInState"`
				Create    map[datatyper.Id]interface{}             `json:"create"`
				Update    map[datatyper.Id][]datatyper.PatchObject `json:"update"`
				Destroy   []datatyper.Id                           `json:"destroy"`

				AccountIdResultRef *ResultReference `json:"#accountId"`
				IfInStateResultRef *ResultReference `json:"#ifInState"`
				CreateResultRef    *ResultReference `json:"#create"`
				UpdateResultRef    *ResultReference `json:"#update"`
				DestroyResultRef   *ResultReference `json:"#destroy"`
			}

			var requestArgs setRequest

			if err := json.Unmarshal(invocation.Arguments, &requestArgs); err != nil {
				if mle, ok := err.(*datatyper.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				//FIXME handle datatype conversion errors properly
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorServerFail()))
				continue
			}

			if len(requestArgs.Create)+len(requestArgs.Update)+len(requestArgs.Destroy) > int(ah.CoreCapabilitySettings.MaxObjectsInSet) {
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorRequestTooLarge()))
				continue
			}

			retAccountId, oldState, newState, created, updated, destroyed, notCreated, notUpdated, notDestroyed, mErr := dtSet.Set(r.Context(), requestArgs.AccountId, requestArgs.IfInState, requestArgs.Create, requestArgs.Update, requestArgs.Destroy)
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
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type copyRequest struct {
				FromAccountId           datatyper.Id                 `json:"fromAccountId"`
				IfFromState             *string                      `json:"ifFromState"`
				AccountId               datatyper.Id                 `json:"accountId"`
				IfInState               *string                      `json:"ifInState"`
				Create                  map[datatyper.Id]interface{} `json:"create"`
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
				if mle, ok := err.(*datatyper.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				//FIXME handle datatype conversion errors properly
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorServerFail()))
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
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type queryRequest struct {
				AccountId            datatyper.Id           `json:"accountId"`
				Filter               *datatyper.Filter      `json:"filter"`
				Sort                 []datatyper.Comparator `json:"sort"`
				Position             datatyper.Int          `json:"position"`
				Anchor               *datatyper.Id          `json:"anchor"`
				AnchorOffset         datatyper.Int          `json:"anchorOffset"`
				Limit                *datatyper.Uint        `json:"limit"`
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
				if mle, ok := err.(*datatyper.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				//FIXME handle datatype conversion errors properly
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorServerFail()))
				continue
			}
			retAccountId, queryState, canCalculateChanges, retPosition, ids, total, retLimit, mErr := dtQuery.Query(r.Context(), requestArgs.AccountId, requestArgs.Filter, requestArgs.Sort, requestArgs.Position, requestArgs.Anchor, requestArgs.AnchorOffset, requestArgs.Limit, requestArgs.CalculateTotal)
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

		case "queryChanges":

			dtQueryChanges, ok := dt.(datatyper.QueryChangeser)
			if !ok {
				//datatype does not have this method
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorUnknownMethod()))
				continue
			}

			type queryChangesRequest struct {
				AccountId       datatyper.Id           `json:"accountId"`
				Filter          *datatyper.Filter      `json:"filter"`
				Sort            []datatyper.Comparator `json:"sort"`
				SinceQueryState string                 `json:"sinceQueryState"`
				MaxChanges      *datatyper.Uint        `json:"maxChanges"`
				UpToId          *datatyper.Id          `json:"upToId"`
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
				if mle, ok := err.(*datatyper.MethodLevelError); ok {
					response.addMethodResponse(invocationResponse.withArgError(mle))
					continue
				}
				//FIXME handle datatype conversion errors properly
				response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorServerFail()))
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
			response.addMethodResponse(invocationResponse.withArgError(datatyper.NewMethodLevelErrorServerFail()))
		}
	}

	response.SessionState = ah.SessionStater.SessionState()
	writeOutput(200, response, w)
}

// writeOutput encodes a the body into json and writes the output the the reponse writer
func writeOutput(statusCode int, body interface{}, w http.ResponseWriter) {

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
	w.Header().Add(HeaderContentType, HeaderContentTypeJSON)
	w.WriteHeader(statusCode)
	w.Write(jsonBytes)
}
