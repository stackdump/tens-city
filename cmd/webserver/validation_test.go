package main

import (
	"testing"
)

func TestValidateJSONLD(t *testing.T) {
	tests := []struct {
		name      string
		doc       map[string]interface{}
		wantError bool
	}{
		{
			name: "Valid JSON-LD with string context",
			doc: map[string]interface{}{
				"@context": "https://schema.org",
				"@type":    "Person",
				"name":     "John Doe",
			},
			wantError: false,
		},
		{
			name: "Valid JSON-LD with object context",
			doc: map[string]interface{}{
				"@context": map[string]interface{}{
					"name": "http://schema.org/name",
				},
				"name": "John Doe",
			},
			wantError: false,
		},
		{
			name: "Valid JSON-LD with array context",
			doc: map[string]interface{}{
				"@context": []interface{}{
					"https://schema.org",
					map[string]interface{}{
						"custom": "http://example.com/custom",
					},
				},
				"@type": "Person",
			},
			wantError: false,
		},
		{
			name: "Missing @context",
			doc: map[string]interface{}{
				"name": "John Doe",
			},
			wantError: true,
		},
		{
			name: "Invalid @context type",
			doc: map[string]interface{}{
				"@context": 123,
				"name":     "John Doe",
			},
			wantError: true,
		},
		{
			name: "Key with control characters",
			doc: map[string]interface{}{
				"@context": "https://schema.org",
				"name\x00": "John Doe",
			},
			wantError: true,
		},
		{
			name: "Excessively deep nesting",
			doc: func() map[string]interface{} {
				// Create a document with 60 levels of nesting (exceeds max of 50)
				doc := map[string]interface{}{
					"@context": "https://schema.org",
				}
				current := doc
				for i := 0; i < 60; i++ {
					nested := map[string]interface{}{}
					current["nested"] = nested
					current = nested
				}
				return doc
			}(),
			wantError: true,
		},
		{
			name: "Reasonable nesting depth",
			doc: func() map[string]interface{} {
				// Create a document with 30 levels of nesting (within max of 50)
				doc := map[string]interface{}{
					"@context": "https://schema.org",
				}
				current := doc
				for i := 0; i < 30; i++ {
					nested := map[string]interface{}{}
					current["nested"] = nested
					current = nested
				}
				return doc
			}(),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONLD(tt.doc)
			if (err != nil) != tt.wantError {
				t.Errorf("validateJSONLD() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateDepth(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		maxDepth  int
		wantError bool
	}{
		{
			name: "Simple object within depth",
			value: map[string]interface{}{
				"key": "value",
			},
			maxDepth:  5,
			wantError: false,
		},
		{
			name: "Nested object exceeds depth",
			value: func() interface{} {
				obj := map[string]interface{}{}
				current := obj
				for i := 0; i < 10; i++ {
					nested := map[string]interface{}{}
					current["nested"] = nested
					current = nested
				}
				return obj
			}(),
			maxDepth:  5,
			wantError: true,
		},
		{
			name: "Array with nested objects",
			value: []interface{}{
				map[string]interface{}{
					"nested": map[string]interface{}{
						"deep": "value",
					},
				},
			},
			maxDepth:  5,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDepth(tt.value, 0, tt.maxDepth)
			if (err != nil) != tt.wantError {
				t.Errorf("validateDepth() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateKeys(t *testing.T) {
	tests := []struct {
		name      string
		doc       map[string]interface{}
		wantError bool
	}{
		{
			name: "Valid keys",
			doc: map[string]interface{}{
				"name":  "John",
				"email": "john@example.com",
			},
			wantError: false,
		},
		{
			name: "Key with null byte",
			doc: map[string]interface{}{
				"name\x00": "John",
			},
			wantError: true,
		},
		{
			name: "Nested object with invalid key",
			doc: map[string]interface{}{
				"user": map[string]interface{}{
					"name\x01": "John",
				},
			},
			wantError: true,
		},
		{
			name: "Array with object containing invalid key",
			doc: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"name\x02": "John",
					},
				},
			},
			wantError: true,
		},
		{
			name: "Keys with allowed whitespace",
			doc: map[string]interface{}{
				"name\twith\ttabs":     "value1",
				"name\nwith\nnewlines": "value2",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeys(tt.doc)
			if (err != nil) != tt.wantError {
				t.Errorf("validateKeys() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
