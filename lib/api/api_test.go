package api

import (
	"testing"
)

// Helper function to create a pointer to an int
func intPtr(i int) *int {
	return &i
}

func TestJsonParse(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		wantErr bool
	}{
		{
			name:    "empty body",
			body:    []byte{},
			wantErr: true,
		},
		{
			name:    "nil body",
			body:    nil,
			wantErr: true,
		},
		{
			name:    "valid JSON",
			body:    []byte(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			body:    []byte(`{"key":`),
			wantErr: true,
		},
		{
			name:    "JSON with unexpected structure",
			body:    []byte(`{"unexpectedKey": {"nestedKey": "nestedValue"}}`),
			wantErr: false, // Assuming APIResponse has a flexible structure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := JsonParse(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("JsonParse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && resp == nil {
				t.Errorf("JsonParse() response = %v, want not nil", resp)
			}
		})
	}
}
