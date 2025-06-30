package util

import (
	"reflect"
	"testing"
)

func TestFields(t *testing.T) {
	tests := []struct {
		name           string
		keysAndValues  []interface{}
		expectedFields map[string]interface{}
	}{
		{
			name:           "empty input",
			keysAndValues:  []interface{}{},
			expectedFields: map[string]interface{}{},
		},
		{
			name:           "single key no value",
			keysAndValues:  []interface{}{"key"},
			expectedFields: map[string]interface{}{},
		},
		{
			name:           "one key-value pair",
			keysAndValues:  []interface{}{"key", "value"},
			expectedFields: map[string]interface{}{"key": "value"},
		},
		{
			name:           "two key-value pairs",
			keysAndValues:  []interface{}{"key1", 123, "key2", true},
			expectedFields: map[string]interface{}{"key1": 123, "key2": true},
		},
		{
			name:           "odd number of elements",
			keysAndValues:  []interface{}{"key1", 123, "key2"},
			expectedFields: map[string]interface{}{"key1": 123},
		},
		{
			name:           "non-string key",
			keysAndValues:  []interface{}{42, "answer"},
			expectedFields: map[string]interface{}{"42": "answer"},
		},
		{
			name:           "mixed key types",
			keysAndValues:  []interface{}{42, "answer", true, "bool"},
			expectedFields: map[string]interface{}{"42": "answer", "true": "bool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fields(tt.keysAndValues)
			if !reflect.DeepEqual(got, tt.expectedFields) {
				t.Errorf("fields(%v) = %v, want %v", tt.keysAndValues, got, tt.expectedFields)
			}
		})
	}
}
