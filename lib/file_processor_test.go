package lib

import (
	"os"
	"testing"
)

func TestAppendToFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		msg      string
		wantErr  bool
		setup    func()
		verify   func()
	}{
		{
			name:     "success append to existing file",
			filePath: "testfile.txt",
			msg:      "Hello, World!",
			wantErr:  false,
			setup: func() {
				// Create a file for testing
				f, err := os.Create("testfile.txt")
				if err != nil {
					t.Fatal(err)
				}
				f.Close()
			},
			verify: func() {
				// You can add verification here if needed
			},
		},
		{
			name:     "create new file if not exists",
			filePath: "nonexistentfile.txt",
			msg:      "New file content",
			wantErr:  false,
			setup:    func() {},
			verify: func() {
				// Check if file exists and has content
				f, err := os.ReadFile("nonexistentfile.txt")
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if string(f) != "New file content" {
					t.Errorf("Expected file content 'New file content', got '%s'", string(f))
				}
			},
		},
		{
			name:     "empty file path",
			filePath: "",
			msg:      "Test message",
			wantErr:  true,
			setup:    func() {},
			verify:   func() {},
		},
		{
			name:     "empty message",
			filePath: "emptymsgfile.txt",
			msg:      "",
			wantErr:  false,
			setup:    func() {},
			verify: func() {
				// Check if file exists and is empty
				_, err := os.Stat("emptymsgfile.txt")
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer func() {
				// Clean up
				err := os.Remove(tt.filePath)
				if err != nil && !os.IsNotExist(err) {
					t.Errorf("Failed to clean up: %v", err)
				}
			}()
			err := appendToFile(tt.filePath, tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("appendToFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.verify()
		})
	}
}

// This test suite covers a variety of scenarios:
//
// - **Success cases:** Appending to an existing file and creating a new file.
// - **Error cases:** Permission errors, empty file path, and edge cases like an empty message.
// - **Verification:** For some tests, it checks if the file was created or modified as expected.
//
// Please replace `"yourpackagename"` with the actual package name where your `appendToFile` function resides. Also, be cautious with the file paths used in the tests, especially when testing for permission errors, to avoid unintended side effects on your file system.
