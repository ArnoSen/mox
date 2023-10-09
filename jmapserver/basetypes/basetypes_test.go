package basetypes

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {

	t.Run("Unmarshal", func(t *testing.T) {

		for _, tc := range []struct {
			Testcase string
			JSON     string
			EError   bool
			EFilter  Filter
		}{
			{
				Testcase: "simple assertion",
				JSON:     `{ "operator": "NOT", "conditions": [ { "id_of_inbox":"abc"} ] }`,
				EFilter: Filter{
					filter: FilterOperator{
						Operator: FilterOperatorTypeNOT,
						Conditions: []interface{}{
							FilterCondition{
								Property:      "id_of_inbox",
								AssertedValue: "abc",
							},
						},
					},
				},
			},
			{
				Testcase: "simple filter condition",
				JSON:     `{ "id_of_inbox":"abc"}`,
				EFilter: Filter{
					filter: FilterCondition{
						Property:      "id_of_inbox",
						AssertedValue: "abc",
					},
				},
			},
		} {
			t.Run(tc.Testcase, func(t *testing.T) {

				var filter Filter

				err := json.Unmarshal([]byte(tc.JSON), &filter)
				if err != nil {
					if !tc.EError {
						t.Fatalf("got error %s but was expecting no error", err)
					}
				} else {
					if !reflect.DeepEqual(filter, tc.EFilter) {
						t.Fatalf("was expecting %s but got %s", tc.EFilter, filter)
					}
				}

			})
		}
	})

}

func TestFilterOperator(t *testing.T) {

	t.Run("Unmarshal", func(t *testing.T) {

		for _, tc := range []struct {
			Testcase        string
			JSON            string
			EError          bool
			EFilterOperator FilterOperator
		}{
			{
				Testcase: "simple assertion",
				JSON:     `{ "operator": "NOT", "conditions": [ { "id_of_inbox":"abc"} ] }`,
				EFilterOperator: FilterOperator{
					Operator: FilterOperatorTypeNOT,
					Conditions: []interface{}{
						FilterCondition{
							Property:      "id_of_inbox",
							AssertedValue: "abc",
						},
					},
				},
			},
			{
				Testcase: "More complex",
				JSON:     `{ "operator": "AND", "conditions": [ { "id_of_inbox":"abc"}, { "operator" : "NOT", "conditions": [ {"sender": "me" }] }] }`,
				EFilterOperator: FilterOperator{
					Operator: FilterOperatorTypeAND,
					Conditions: []interface{}{
						FilterCondition{
							Property:      "id_of_inbox",
							AssertedValue: "abc",
						},
						FilterOperator{
							Operator: FilterOperatorTypeNOT,
							Conditions: Conditions{
								FilterCondition{
									Property:      "sender",
									AssertedValue: "me",
								},
							},
						},
					},
				},
			},
		} {
			t.Run(tc.Testcase, func(t *testing.T) {

				var filterOperator FilterOperator

				err := json.Unmarshal([]byte(tc.JSON), &filterOperator)
				if err != nil {
					if !tc.EError {
						t.Fatalf("got error %s but was expecting no error", err)
					}
				} else {
					if !reflect.DeepEqual(filterOperator, tc.EFilterOperator) {
						t.Fatalf("was expecting %s but got %s", tc.EFilterOperator, filterOperator)
					}
				}

			})
		}
	})

}

func TestFilterCondition(t *testing.T) {
	t.Run("Unmarshal", func(t *testing.T) {

		for _, tc := range []struct {
			Testcase         string
			JSON             string
			EError           bool
			EFilterCondition FilterCondition
		}{
			{
				Testcase: "simple assertion",
				JSON:     `{ "inMailbox": "id_of_inbox" }`,
				EFilterCondition: FilterCondition{
					Property:      "inMailbox",
					AssertedValue: "id_of_inbox",
				},
			},
			{
				Testcase: "Double assertion",
				JSON:     `{ "inMailbox": "id_of_inbox", "other": 1 }`,
				EFilterCondition: FilterCondition{
					Property:      "inMailbox",
					AssertedValue: "id_of_inbox",
				},
				EError: true,
			},
			{
				Testcase: "simple assertion with integer",
				JSON:     `{ "inMailbox": true }`,
				EFilterCondition: FilterCondition{
					Property:      "inMailbox",
					AssertedValue: true,
				},
			},
		} {
			t.Run(tc.Testcase, func(t *testing.T) {

				var filterCondition FilterCondition

				err := json.Unmarshal([]byte(tc.JSON), &filterCondition)
				if err != nil {
					if !tc.EError {
						t.Fatalf("got error %s but was expecting no error", err)
					}
				} else {
					if filterCondition != tc.EFilterCondition {
						t.Fatalf("was expecting %s but got %s", tc.EFilterCondition, filterCondition)
					}
				}

			})
		}
	})
}
