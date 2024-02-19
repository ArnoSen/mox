package httphandler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// JSONProblem conforms to https://datatracker.ietf.org/doc/html/rfc7807
type JSONProblem struct {
	Title   string `json:"title"`
	Details string `json:"details,omitempty"`
}

func (jp JSONProblem) Error() string {
	return fmt.Sprintf("err %s%s", jp.Title, func(detail string) string {
		if detail == "" {
			return ""
		}
		return " (" + detail + ")"
	}(jp.Details))
}

var TypeCannotBeEmpty = JSONProblem{
	Title:   "type is empty",
	Details: "type cannot be empty",
}

var UnknownAccount = JSONProblem{
	Title: "unknow account",
}

func sendUserErr(w http.ResponseWriter, err JSONProblem) {
	w.WriteHeader(http.StatusBadRequest)

	if err := json.NewEncoder(w).Encode(err); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		//have an hardcoded fallback if marshalling fails
		w.Write([]byte(fmt.Sprintf(`{ "title": "internal server error", "details": %q}`, err.Error())))
	}
}
