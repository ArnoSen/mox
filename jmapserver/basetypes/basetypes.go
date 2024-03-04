package basetypes

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
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

func NewIdFromInt64(i int64) Id {
	return Id(fmt.Sprintf("%d", i))
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

// Int64 returns an int64 if the format is suitable. If not, an error is sent
func (id Id) Int64() (int64, error) {
	return strconv.ParseInt(string(id), 10, 64)
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

func (u Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(u).Format(time.RFC3339))

}

// https://datatracker.ietf.org/doc/html/rfc8620#section-1.4
type UTCDate time.Time

func (u UTCDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(u).UTC().Format(time.RFC3339))

}

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

// FIXME this should return a method level error because that is their only scope
func ParseUint(i int64) (Uint, *mlevelerrors.MethodLevelError) {
	if i < 0 || float64(i) > (math.Pow(2, 53)-1) {
		return Uint(0), mlevelerrors.NewMethodLevelErrorInvalidArguments(fmt.Sprintf("uint out of range"))
	}
	return Uint(uint64(i)), nil

}

type FilterOperatorType string

func (fot *FilterOperatorType) UnmarshalJSON(b []byte) error {
	var temp string

	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	switch temp {
	case string(FilterOperatorTypeAND):
		*fot = FilterOperatorTypeAND
	case string(FilterOperatorTypeOR):
		*fot = FilterOperatorTypeOR
	case string(FilterOperatorTypeNOT):
		*fot = FilterOperatorTypeNOT
	default:
		return fmt.Errorf("empty or unknown operator type")
	}

	return nil
}

const (
	FilterOperatorTypeAND FilterOperatorType = "AND"
	FilterOperatorTypeOR  FilterOperatorType = "OR"
	FilterOperatorTypeNOT FilterOperatorType = "NOT"
)

/*
some examples for filter

"filter": {
    "operator": "OR",
    "conditions": [
      { "hasKeyword": "music" },
      { "hasKeyword": "video" }
    ]
  },

"filter": { "inMailbox": "id_of_inbox" },
*/

//FIXME need to look into generics here because it could simply things, i guess...

type Filter struct {
	//I need a structure like this because otherwise the custom unmarshal method (like all other methods) cannot have a receiver of type interface
	filter interface{}
}

func (f Filter) GetFilter() interface{} {
	return f.filter
}

func (fo *Filter) UnmarshalJSON(b []byte) error {

	var tryFilterCondition FilterCondition
	if err := json.Unmarshal(b, &tryFilterCondition); err == nil {
		if tryFilterCondition.AssertedValue != nil && tryFilterCondition.Property != "" {
			//we have a valid filter
			*fo = Filter{
				filter: tryFilterCondition,
			}
			return nil
		}
	}

	var tryFilterOperator FilterOperator
	if err := json.Unmarshal(b, &tryFilterOperator); err == nil {
		//we have a valid filter operator
		*fo = Filter{
			filter: tryFilterOperator,
		}
		return nil
	}

	return &json.UnmarshalFieldError{
		Key:  "filter",
		Type: reflect.TypeOf(fo),
	}
}

// FilterOperator is a filter containing an operator and a set of conditions
type FilterOperator struct {
	Operator FilterOperatorType `json:"operator"`

	//probably needs some generics here
	Conditions Conditions `json:"conditions"`
}

func (fo *FilterOperator) UnmarshalJSON(b []byte) error {
	//I need this supporting type to not get into a loop
	type foCopy struct {
		Operator FilterOperatorType `json:"operator"`

		//probably needs some generics here
		Conditions Conditions `json:"conditions"`
	}
	var temp foCopy

	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	if temp.Operator == FilterOperatorTypeNOT && len(temp.Conditions) != 1 {
		return fmt.Errorf("when using not, there can only be one condition")
	}

	if (temp.Operator == FilterOperatorTypeOR || temp.Operator == FilterOperatorTypeAND) && len(temp.Conditions) < 2 {
		return fmt.Errorf("when using and/or, there must be at least 2 conditions")
	}

	*fo = FilterOperator(temp)

	return nil
}

type Conditions []interface{}

func (cos *Conditions) UnmarshalJSON(b []byte) error {
	var temp []json.RawMessage

	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	var result []interface{}

	for _, conditionJSON := range temp {
		var tryFilterCondition FilterCondition
		if err := json.Unmarshal(conditionJSON, &tryFilterCondition); err == nil {
			if tryFilterCondition.Property != "" && tryFilterCondition.AssertedValue != nil {
				//we have a match
				result = append(result, tryFilterCondition)
				continue
			}
		}

		var tryFilterOperator FilterOperator
		if err := json.Unmarshal(conditionJSON, &tryFilterOperator); err == nil {
			if tryFilterOperator.Operator != "" {
				result = append(result, tryFilterOperator)
				continue
			}
		}
		return fmt.Errorf("invalid conditions format")
	}

	*cos = result

	return nil
}

type FilterCondition struct {
	Property      string
	AssertedValue interface{}
}

func (fc *FilterCondition) UnmarshalJSON(b []byte) error {

	var stringMap map[string]interface{}

	if err := json.Unmarshal(b, &stringMap); err != nil {
		return err
	}

	if len(stringMap) != 1 {
		return fmt.Errorf("invalid format for FilterCondition")
	}
	for k, v := range stringMap {
		*fc = FilterCondition{
			Property:      k,
			AssertedValue: v,
		}
	}
	return nil
}

type Comparator struct {
	//The name of the property on the Foo objects to compare.
	Property string

	IsAscending bool

	//The identifier, as registered in the collation registry defined in [RFC4790]
	Collation string
}

type PatchObject map[string]interface{}
