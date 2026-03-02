package dlog_test

import (
	"testing"

	"github.com/rohmanhakim/dlog"
	"github.com/stretchr/testify/assert"
)

func TestFilterFields(t *testing.T) {
	tests := []struct {
		name          string
		entry         map[string]any
		includeFields []string
		excludeFields []string
		expected      map[string]any
	}{
		{
			name: "no filtering - empty include and exclude",
			entry: map[string]any{
				"field1": "value1",
				"field2": "value2",
				"field3": "value3",
			},
			includeFields: []string{},
			excludeFields: []string{},
			expected: map[string]any{
				"field1": "value1",
				"field2": "value2",
				"field3": "value3",
			},
		},
		{
			name: "no filtering - nil include and exclude",
			entry: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
			includeFields: nil,
			excludeFields: nil,
			expected: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
		},
		{
			name: "include only specific fields",
			entry: map[string]any{
				"field1": "value1",
				"field2": "value2",
				"field3": "value3",
			},
			includeFields: []string{"field1", "field3"},
			excludeFields: []string{},
			expected: map[string]any{
				"field1": "value1",
				"field3": "value3",
			},
		},
		{
			name: "exclude specific fields",
			entry: map[string]any{
				"field1": "value1",
				"field2": "value2",
				"field3": "value3",
			},
			includeFields: []string{},
			excludeFields: []string{"field2"},
			expected: map[string]any{
				"field1": "value1",
				"field3": "value3",
			},
		},
		{
			name: "include and exclude combined - exclude takes precedence",
			entry: map[string]any{
				"field1": "value1",
				"field2": "value2",
				"field3": "value3",
			},
			includeFields: []string{"field1", "field2", "field3"},
			excludeFields: []string{"field2"},
			expected: map[string]any{
				"field1": "value1",
				"field3": "value3",
			},
		},
		{
			name: "exclude all fields",
			entry: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
			includeFields: []string{},
			excludeFields: []string{"field1", "field2"},
			expected:      map[string]any{},
		},
		{
			name: "include non-existent field",
			entry: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
			includeFields: []string{"field3"},
			excludeFields: []string{},
			expected:      map[string]any{},
		},
		{
			name: "exclude non-existent field",
			entry: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
			includeFields: []string{},
			excludeFields: []string{"field3"},
			expected: map[string]any{
				"field1": "value1",
				"field2": "value2",
			},
		},
		{
			name: "entry with nested values",
			entry: map[string]any{
				"string_field": "value",
				"int_field":    42,
				"bool_field":   true,
				"nested_field": map[string]any{
					"nested_key": "nested_value",
				},
			},
			includeFields: []string{"string_field", "int_field", "nested_field"},
			excludeFields: []string{},
			expected: map[string]any{
				"string_field": "value",
				"int_field":    42,
				"nested_field": map[string]any{
					"nested_key": "nested_value",
				},
			},
		},
		{
			name:          "empty entry",
			entry:         map[string]any{},
			includeFields: []string{"field1"},
			excludeFields: []string{},
			expected:      map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dlog.FilterFields(tt.entry, tt.includeFields, tt.excludeFields)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterFields_DoesNotModifyOriginal(t *testing.T) {
	original := map[string]any{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
	}

	// Apply filtering
	result := dlog.FilterFields(original, []string{}, []string{"field2"})

	// Original should not be modified
	assert.Equal(t, map[string]any{
		"field1": "value1",
		"field2": "value2",
		"field3": "value3",
	}, original, "original map should not be modified")

	// Result should be filtered
	assert.Equal(t, map[string]any{
		"field1": "value1",
		"field3": "value3",
	}, result, "result should be filtered")
}
