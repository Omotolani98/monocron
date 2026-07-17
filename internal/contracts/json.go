package contracts

import "encoding/json"

// MarshalJSON returns the JSON encoding of v.
func MarshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalJSON parses JSON-encoded data into v.
func UnmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
