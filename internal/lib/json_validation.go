package lib

import (
	"context"
	"encoding/json"

	"github.com/qri-io/jsonschema"
)

// ValidateJSON validates a JSON raw message against a given JSON schema.
// It returns a list of validation errors if the JSON is invalid.
func ValidateJSON(content json.RawMessage, schemaString string) ([]jsonschema.KeyError, error) {
	rs := &jsonschema.Schema{}
	if err := json.Unmarshal([]byte(schemaString), rs); err != nil {
		return nil, err
	}

	return rs.ValidateBytes(context.Background(), content)
}

// ExamplePostContentSchema returns an example JSON schema for post content.
// In a real application, this might be loaded from a configuration file or database.
func ExamplePostContentSchema() string {
	return `{
		"type": "object",
		"properties": {
			"type": {"type": "string", "enum": ["text", "image", "video"]},
			"text": {"type": "string"},
			"url": {"type": "string", "format": "uri"},
			"caption": {"type": "string"}
		},
		"required": ["type"],
		"if": {
			"properties": { "type": { "const": "text" } }
		},
		"then": {
			"required": ["text"]
		},
		"if": {
			"properties": { "type": { "const": "image" } }
		},
		"then": {
			"required": ["url"]
		},
		"if": {
			"properties": { "type": { "const": "video" } }
		},
		"then": {
			"required": ["url"]
		}
	}`
}
