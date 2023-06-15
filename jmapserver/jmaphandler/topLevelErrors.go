package jmaphandler

import (
	"fmt"
	"net/http"
)

type RequestLevelErrorType string

const (
	RequestLevelErrorTypeUnknownCapability = "urn:ietf:params:jmap:error:unknownCapability"
	RequestLevelErrorTypeNotJSON           = "urn:ietf:params:jmap:error:notJSON"
	RequestLevelErrorTypeNotRequest        = "urn:ietf:params:jmap:error:notRequest"
	RequestLevelErrorTypeCapabilityLimit   = "urn:ietf:params:jmap:error:limit"
)

type LimitType string

var (
	LimitTypeMaxSizeRequest        LimitType = "maxSizeRequest"
	LimitTypeMaxSizeUpload         LimitType = "maxSizeUpload"
	LimitTypeMaxConcurrentUpload   LimitType = "maxConcurrentUpload"
	LimitTypeMaxConcurrentRequests LimitType = "maxConcurrentRequests"
	LimitTypeMaxCallsInRequest     LimitType = "maxCallsInRequest"
	LimitTypeMaxObjectsInSet       LimitType = "maxObjectsInSet"
)

type RequestLevelError struct {
	Type   string     `json:"type"`
	Status int        `json:"status"`
	Detail string     `json:"detail"`
	Limit  *LimitType `json:"limit,omitempty"`
}

func (rle RequestLevelError) Error() string {
	return fmt.Sprintf("%s: %s", rle.Type, rle.Detail)
}

func NewRequestLevelErrorUnknownCapability(detail string) RequestLevelError {
	return RequestLevelError{
		Type:   RequestLevelErrorTypeUnknownCapability,
		Status: http.StatusBadRequest,
		Detail: detail,
	}
}

func NewRequestLevelErrorNotJSONContentType() RequestLevelError {
	return RequestLevelError{
		Type:   RequestLevelErrorTypeNotJSON,
		Status: http.StatusBadRequest,
		Detail: "the content type of the request is not application/json",
	}
}

func NewRequestLevelErrorNotJSON(detail string) RequestLevelError {
	return RequestLevelError{
		Type:   RequestLevelErrorTypeNotJSON,
		Status: http.StatusBadRequest,
		Detail: detail,
	}
}

func NewRequestLevelErrorNotRequest(detail string) RequestLevelError {
	return RequestLevelError{
		Type:   RequestLevelErrorTypeNotRequest,
		Status: http.StatusBadRequest,
		Detail: detail,
	}
}

func NewRequestLevelErrorCapabilityLimit(limitType LimitType, detail string) RequestLevelError {
	return RequestLevelError{
		Limit:  &limitType,
		Type:   RequestLevelErrorTypeCapabilityLimit,
		Status: http.StatusBadRequest,
		Detail: detail,
	}
}
