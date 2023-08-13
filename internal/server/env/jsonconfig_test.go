package env

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetJSONData(t *testing.T) {
	tests := []struct {
		name      string
		fp        string
		wantError bool
	}{
		{
			name:      "Test 1. Correct json config",
			fp:        "test_configs/all_fields_set.json",
			wantError: false,
		},
		{
			name:      "Test 2. Json config file is not exist",
			fp:        "test_configs/not_exist.json",
			wantError: true,
		},
		{
			name:      "Test 3. Json config file is not valid",
			fp:        "test_configs/json_error.json",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getJSONData(tt.fp)
			assert.Equal(t, tt.wantError, err != nil)
		})
	}
}
