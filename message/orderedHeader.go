package message

import "net/textproto"

// OrderedHeader represents headers in there original order
type OrderedHeader []KV

// KV is a key-value pair
type KV struct {
	Name, Value string
}

// Last returns the value of the last occurence of a particular header. This is a JMAP requirement
func (ohs OrderedHeader) Last(header string) string {
	var result string
	for i := len(ohs) - 1; i >= 0; i-- {
		if ohs[i].Name == header {
			return ohs[i].Value
		}
	}
	return result
}

// Values returns all the values for header
func (ohs OrderedHeader) Values(header string) []string {
	var result []string
	for _, oh := range ohs {
		if oh.Name == header {
			//because we need the last value, we do not break out but wait for the last result
			result = append(result, oh.Value)
		}
	}
	return result
}

// MIMEHeader returns a MIMEHeader object. This adaptor is there to use some methods that are defined for MIMEHeader
func (ohs OrderedHeader) MIMEHeader() textproto.MIMEHeader {
	result := textproto.MIMEHeader{}
	for _, oh := range ohs {
		if _, exists := result[oh.Name]; exists {
			result[oh.Name] = append(result[oh.Name], oh.Value)
		}

	}
	return result
}
