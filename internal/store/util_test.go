package store

import (
	"reflect"
	"testing"
)

func TestFilterMap(t *testing.T) {
	tests := []struct {
		name     string
		inputMap map[string]int
		keys     []string
		mode     filterMode
		expected map[string]int
	}{
		{
			name: "include keys",
			inputMap: map[string]int{
				"key1": 1,
				"key2": 2,
				"key3": 3,
			},
			keys: []string{"key1", "key3"},
			mode: include,
			expected: map[string]int{
				"key1": 1,
				"key3": 3,
			},
		},
		{
			name: "exclude keys",
			inputMap: map[string]int{
				"key1": 1,
				"key2": 2,
				"key3": 3,
			},
			keys: []string{"key1", "key3"},
			mode: exclude,
			expected: map[string]int{
				"key2": 2,
			},
		},
		{
			name: "include no keys",
			inputMap: map[string]int{
				"key1": 1,
				"key2": 2,
				"key3": 3,
			},
			keys:     []string{},
			mode:     include,
			expected: map[string]int{},
		},
		{
			name: "exclude no keys",
			inputMap: map[string]int{
				"key1": 1,
				"key2": 2,
				"key3": 3,
			},
			keys: []string{},
			mode: exclude,
			expected: map[string]int{
				"key1": 1,
				"key2": 2,
				"key3": 3,
			},
		},
		{
			name: "exclude keys match",
			inputMap: map[string]int{
				"key1": 1,
				"key2": 2,
				"key3": 3,
			},
			keys:     []string{"key"},
			mode:     exclude,
			expected: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterMap(tt.inputMap, tt.keys, tt.mode)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterMap() = %v, want %v", result, tt.expected)
			}
		})
	}
}
