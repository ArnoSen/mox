package datatyper

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type PatchObject map[string]interface{}

type SetError struct {
	Type        ErrorType
	Description *string
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

type AddedItem struct {
	Id    Id
	Index Uint
}

type Echoer interface {
	Echo(ctx context.Context, content json.RawMessage) (resp map[string]interface{}, mErr *MethodLevelError)
}

type Getter interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
	Get(ctx context.Context, accountId Id, ids []Id, properties []string) (retAccountId Id, state string, list []interface{}, notFound []Id, mErr *MethodLevelError)
}

type Changeser interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.2
	Changes(ctx context.Context, accountId Id, sinceState string, maxChanges *Uint) (retAccountId Id, oldState, newState string, hasMoreChanges bool, created, updated, destroyed []Id, mErr *MethodLevelError)
}

type Setter interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.3
	Set(ctx context.Context, accountId Id, ifInState *string, create map[Id]interface{}, update map[Id][]PatchObject, destroy []Id) (retAccountId Id, oldState *string, newState string, created, updated, destroyed map[Id]interface{}, notCreated, notUpdated, notDestroyed map[Id]SetError, mErr *MethodLevelError)
}

type Copier interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.4
	Copy(ctx context.Context, fromAccountId Id, ifFromState *string, accountId Id, ifInState *string, create map[Id]interface{}, onSuccessDestroyOriginal bool, destroyFromIfInState *string) (retFromAccountId, retAccountId Id, oldState *string, newState string, created map[Id]interface{}, notCreated map[Id]SetError, mErr *MethodLevelError)
}

type Querier interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.5
	Query(ctx context.Context, accountId Id, filter *Filter, sort []Comparator, position Int, anchor *Id, anchorOffset Int, limit *Uint, calculateTotal bool) (retAccountId Id, queryState string, canCalculateChanges bool, retPosition Int, ids []Id, total Uint, retLimit Uint, mErr *MethodLevelError)
}

type QueryChangeser interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.6
	QueryChanges(ctx context.Context, accountId Id, filter *Filter, sort []Comparator, sinceQueryState string, maxChanges *Uint, upToId *Id, calculateTotal bool) (retAccountId Id, oldQueryState, newQueryState string, total Uint, removed []Id, added []AddedItem, mErr *MethodLevelError)
}
