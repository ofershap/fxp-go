package fxp

import (
	"bytes"
	"encoding/json"
)

// MarshalJSON renders output with HTML escaping disabled, matching
// JSON.stringify byte output for differential comparison.
func MarshalJSON(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}
