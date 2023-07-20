package basetypes

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

// https://datatracker.ietf.org/doc/html/rfc8620#section-1.2
type Id string

// ParseId parses an id from string
func ParseId(idStr string) (Id, *mlevelerrors.MethodLevelError) {
	if !regexp.MustCompile("^[A-Za-z0-9-_]{1,255}?$").MatchString(idStr) {
		return Id(""), mlevelerrors.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("invalid id %s", idStr))
	}
	return Id(idStr), nil
}

func (id *Id) UnmarshalJSON(b []byte) error {
	var idStr string

	if err := json.Unmarshal(b, &idStr); err != nil {
		return err
	}

	if idStr == "" {
		return mlevelerrors.NewMethodLevelErrorInvalidArguments("accountId cannot be empty")
	}

	newId, mlErr := ParseId(idStr)
	if mlErr != nil {
		return mlErr
	}

	*id = newId
	return nil
}

func (id Id) IsEmpty() bool {
	return len(id) == 0
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-1.3
type Uint uint64

func (ui *Uint) UnmarshalJSON(b []byte) error {
	var uiInt64 int64

	if err := json.Unmarshal(b, &uiInt64); err != nil {
		return err
	}

	newUi, mlErr := ParseUint(uiInt64)
	if mlErr != nil {
		return mlErr
	}

	*ui = newUi
	return nil
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-1.3
type Int int64

// https://datatracker.ietf.org/doc/html/rfc8620#section-1.4
type Date time.Time

// https://datatracker.ietf.org/doc/html/rfc8620#section-1.4
type UTCDate time.Time

// ParseIds parses a slice of strings into a slice of Id. If one element fails the parse, an error is returned and the failedId is returned in the response
func ParseIds(idStrs []string) (result []Id, failedId string, mErr *mlevelerrors.MethodLevelError) {
	for _, idStr := range idStrs {
		id, err := ParseId(idStr)
		if err != nil {
			return nil, idStr, err
		}
		result = append(result, id)
	}
	return result, "", nil
}

// FIXME this should return a method level error because that is there only scope
func ParseUint(i int64) (Uint, *mlevelerrors.MethodLevelError) {
	if i < 0 || float64(i) > (math.Pow(2, 53)-1) {
		return Uint(0), mlevelerrors.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("uint out of range"))
	}
	return Uint(uint64(i)), nil

}
