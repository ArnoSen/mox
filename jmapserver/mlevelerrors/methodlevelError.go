package mlevelerrors

import "fmt"

type MethodLevelError struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

func (mle MethodLevelError) Error() string {
	return fmt.Sprintf("methodlevel error type %s: %s", mle.Type, mle.Description)
}

func NewMethodLevelErrorServerPartialFail() *MethodLevelError {
	return &MethodLevelError{
		Type: "serverPartialFail",
	}
}

func NewMethodLevelErrorServerFail() *MethodLevelError {
	return &MethodLevelError{
		Type: "serverFail",
	}
}

func NewMethodLevelErrorUnknownMethod() *MethodLevelError {
	return &MethodLevelError{
		Type: "unknownMethod",
	}
}

func NewMethodLevelErrorInvalidArguments(description string) *MethodLevelError {
	return &MethodLevelError{
		Type:        "invalidArguments",
		Description: description,
	}
}

func NewMethodLevelErrorInvalidResultReference(description string) *MethodLevelError {
	return &MethodLevelError{
		Type:        "invalidResultReference",
		Description: description,
	}
}

func NewMethodLevelErrorForbidden() *MethodLevelError {
	return &MethodLevelError{
		Type: "forbidden",
	}
}

func NewMethodLevelErrorAccountForFound() *MethodLevelError {
	return &MethodLevelError{
		Type: "accountNotFound",
	}
}

func NewMethodLevelErrorAccountNotSupportedByMethod() *MethodLevelError {
	return &MethodLevelError{
		Type: "accountNotSupportedByMethod",
	}
}

func NewMethodLevelErrorAccountReadOnly() *MethodLevelError {
	return &MethodLevelError{
		Type: "accountReadOnly",
	}
}

func NewMethodLevelErrorRequestTooLarge() *MethodLevelError {
	return &MethodLevelError{
		Type: "requestTooLarge",
	}
}

type ErrorType string

type SetError struct {
	Type        ErrorType
	Description *string
}
