// Package jsonflex reads the fields a storefront types more than one way.
package jsonflex

import "encoding/json"

// String reads a field the storefront types differently by game: set is a
// plain code for most lines but a whole object for some Pokemon products, of
// which only the id names the set. Anything else decodes to empty rather than
// failing the page it arrived on.
type String string

// UnmarshalJSON implements json.Unmarshaler.
func (f *String) UnmarshalJSON(data []byte) error {
	var plain string
	if json.Unmarshal(data, &plain) == nil {
		*f = String(plain)
		return nil
	}
	var object struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(data, &object) == nil {
		*f = String(object.ID)
		return nil
	}
	*f = ""
	return nil
}
