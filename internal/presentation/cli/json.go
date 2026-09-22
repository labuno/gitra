package cli

import (
	"encoding/json"
	"io"
)

// schemaVersion is the frozen JSON contract version (addendum appendix A.1).
const schemaVersion = 1

// WriteJSON renders one payload with the frozen conventions: two-space
// indentation, HTML escaping disabled, trailing newline.
func WriteJSON(w io.Writer, payload any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(payload)
}
