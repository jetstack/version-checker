package leveledlogger

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLogger_fields(t *testing.T) {
	tests := []struct {
		name           string
		keysAndValues  []interface{}
		expectedFields map[string]interface{}
		description    string
	}{
		{
			name:           "empty slice",
			keysAndValues:  []interface{}{},
			expectedFields: nil,
			description:    "should return original entry when no keys and values provided",
		},
		{
			name:           "nil slice",
			keysAndValues:  nil,
			expectedFields: nil,
			description:    "should return original entry when nil keys and values provided",
		},
		{
			name:          "single key-value pair",
			keysAndValues: []interface{}{"key1", "value1"},
			expectedFields: map[string]interface{}{
				"key1": "value1",
			},
			description: "should correctly handle single key-value pair",
		},
		{
			name:          "multiple key-value pairs",
			keysAndValues: []interface{}{"key1", "value1", "key2", "value2", "key3", "value3"},
			expectedFields: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			description: "should correctly handle multiple key-value pairs",
		},
		{
			name:          "odd number of elements - trailing key",
			keysAndValues: []interface{}{"key1", "value1", "key2"},
			expectedFields: map[string]interface{}{
				"key1": "value1",
				"key2": "(MISSING)",
			},
			description: "should add (MISSING) for trailing key without value",
		},
		{
			name:          "single trailing key",
			keysAndValues: []interface{}{"lonely_key"},
			expectedFields: map[string]interface{}{
				"lonely_key": "(MISSING)",
			},
			description: "should handle single trailing key",
		},
		{
			name:          "non-string key at start",
			keysAndValues: []interface{}{123, "value1", "key2", "value2"},
			expectedFields: map[string]interface{}{
				"key2": "value2",
			},
			description: "should skip non-string keys",
		},
		{
			name:          "non-string key in middle",
			keysAndValues: []interface{}{"key1", "value1", 456, "value2", "key3", "value3"},
			expectedFields: map[string]interface{}{
				"key1": "value1",
				"key3": "value3",
			},
			description: "should skip non-string keys in middle",
		},
		{
			name:           "all non-string keys",
			keysAndValues:  []interface{}{123, "value1", 456, "value2"},
			expectedFields: nil,
			description:    "should return original entry when all keys are non-string",
		},
		{
			name:          "non-string trailing key",
			keysAndValues: []interface{}{"key1", "value1", 789},
			expectedFields: map[string]interface{}{
				"key1": "value1",
			},
			description: "should skip non-string trailing key",
		},
		{
			name:          "various value types",
			keysAndValues: []interface{}{"string", "value", "int", 42, "bool", true, "float", 3.14, "nil", nil},
			expectedFields: map[string]interface{}{
				"string": "value",
				"int":    42,
				"bool":   true,
				"float":  3.14,
				"nil":    nil,
			},
			description: "should handle various value types",
		},
		{
			name:          "empty string key",
			keysAndValues: []interface{}{"", "value1", "key2", "value2"},
			expectedFields: map[string]interface{}{
				"":     "value1",
				"key2": "value2",
			},
			description: "should accept empty string as key",
		},
		{
			name:          "mixed valid and invalid keys",
			keysAndValues: []interface{}{"valid1", "value1", 123, "skipped", "valid2", "value2", true, "alsoSkipped", "valid3", "value3"},
			expectedFields: map[string]interface{}{
				"valid1": "value1",
				"valid2": "value2",
				"valid3": "value3",
			},
			description: "should only process valid string keys",
		},
		{
			name:          "duplicate keys - last wins",
			keysAndValues: []interface{}{"key", "first", "key", "second", "key", "third"},
			expectedFields: map[string]interface{}{
				"key": "third",
			},
			description: "should use last value when key is duplicated",
		},
		{
			name:          "struct value",
			keysAndValues: []interface{}{"struct", struct{ Name string }{"test"}},
			expectedFields: map[string]interface{}{
				"struct": struct{ Name string }{"test"},
			},
			description: "should handle struct values",
		},
		{
			name:          "slice value",
			keysAndValues: []interface{}{"slice", []string{"a", "b", "c"}},
			expectedFields: map[string]interface{}{
				"slice": []string{"a", "b", "c"},
			},
			description: "should handle slice values",
		},
		{
			name:          "map value",
			keysAndValues: []interface{}{"map", map[string]int{"a": 1, "b": 2}},
			expectedFields: map[string]interface{}{
				"map": map[string]int{"a": 1, "b": 2},
			},
			description: "should handle map values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base logger
			baseLogger := logrus.New()
			baseEntry := logrus.NewEntry(baseLogger)

			logger := &Logger{
				Entry: baseEntry,
			}

			// Call fields method
			result := logger.fields(tt.keysAndValues)

			// If no fields expected, should return original entry
			if tt.expectedFields == nil {
				assert.Equal(t, baseEntry, result, tt.description)
				return
			}

			// Check that returned entry has expected fields
			assert.NotEqual(t, baseEntry, result, "should return new entry with fields")

			// Verify all expected fields are present
			for key, expectedValue := range tt.expectedFields {
				actualValue, exists := result.Data[key]
				assert.True(t, exists, "field %s should exist", key)
				assert.Equal(t, expectedValue, actualValue, "field %s should have correct value", key)
			}

			// Verify no unexpected fields
			assert.Equal(t, len(tt.expectedFields), len(result.Data), "should have exact number of expected fields")
		})
	}
}
