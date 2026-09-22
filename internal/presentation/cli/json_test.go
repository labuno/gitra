package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteJSONContract(t *testing.T) {
	var buf bytes.Buffer
	payload := struct {
		SchemaVersion int      `json:"schema_version"`
		Accounts      []string `json:"accounts"`
	}{SchemaVersion: schemaVersion, Accounts: []string{}}
	if err := WriteJSON(&buf, payload); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if decoded["schema_version"] != float64(1) {
		t.Fatalf("schema_version = %v, want 1", decoded["schema_version"])
	}
	if buf.String()[len(buf.String())-1] != '\n' {
		t.Fatal("JSON output must end with a newline")
	}
}

func TestWriteJSONNeverNullsSlices(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, struct {
		SchemaVersion int      `json:"schema_version"`
		Failures      []string `json:"failures"`
	}{SchemaVersion: schemaVersion, Failures: []string{}}); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf.Bytes(), []byte("null")) {
		t.Fatalf("empty slices must encode as []: %s", buf.String())
	}
}
