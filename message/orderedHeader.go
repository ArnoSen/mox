package message

import "net/textproto"

type OrderedHeaders []Header

type Header struct {
	Name, Value string
}

func (ohs OrderedHeaders) Last(header string) string {
	var result string
	for i := len(ohs) - 1; i >= 0; i-- {
		if ohs[i].Name == header {
			return ohs[i].Value
		}
	}
	return result
}

func (ohs OrderedHeaders) Values(header string) []string {
	var result []string
	for _, oh := range ohs {
		if oh.Name == header {
			//because we need the last value, we do not break out but wait for the last result
			result = append(result, oh.Value)
		}
	}
	return result
}

func (ohs OrderedHeaders) MIMEHeader() textproto.MIMEHeader {
	result := textproto.MIMEHeader{}

	for _, oh := range ohs {

		if _, exists := result[oh.Name]; exists {
			result[oh.Name] = append(result[oh.Name], oh.Value)
		}

	}
	return result
}
