package input

import (
	"testing"
)

func TestParseOrderType(t *testing.T) {

	tests := []struct {
		name     string
		filePath string
		expected OrderType
	}{
		// --- Happy Path Cases (Valid Endings) ---
		{"Valid S Ending", "path/to/222-some-file-S", 出庫},
		{"Valid K Ending", "another/dir/222-222-data-K", 購入},
		{"Valid G Ending", "just-a-name-G", 外注},
		{"Valid S Ending - No Path", "file-tbd-20-S-2", 出庫},
		{"Valid K Ending - No Path", "doc-123--K-1", 購入},
		{"Valid G Ending - No Path", "2002-1234-tbd-G-その3", 外注},
		{"Valid S Ending - Multiple Hyphens", "prefix-middle-suffix-S", 出庫},
		{"Valid K Ending - Multiple Hyphens", "a-b-c-K", 購入},
		{"Valid G Ending - Multiple Hyphens", "x-y-z-G", 外注},
		{"Valid S Ending - Hyphen at Start", "202-231-tbd-S", 出庫}, // filepath.Base is "-file-S", split is ["", "file", "S"]

		// --- Unhappy Path Cases (Invalid Endings / Default) ---
		{"Invalid Ending - X", "file-X", 未選択},
		{"Invalid Ending - ABC", "data-abc", 未選択},
		{"Invalid Ending - Number", "report-123", 未選択},
		{"Invalid Ending - Empty String after Hyphen", "file-", 未選択}, // Split results in ["file", ""], last is ""
		{"Invalid Ending - Lowercase s", "file--s", 未選択},             // Case sensitive
		{"Invalid Ending - Lowercase k", "file---k", 未選択},
		{"Invalid Ending - Lowercase g", "file--g", 未選択},
		{"Invalid Ending - Mixed Case", "file--S ", 未選択}, // Trailing space

		// --- Edge Cases (No Hyphens, Empty, etc.) ---
		{"No Hyphens - Just S", "S", 未選択}, // Last block is "S", but not after a hyphen
		{"No Hyphens - Just K", "K", 未選択},
		{"No Hyphens - Just G", "G", 未選択},
		{"No Hyphens - Regular Filename", "myfile.txt", 未選択},       // Last block is "myfile.txt"
		{"No Hyphens - Filename with Dots", "archive.tar.gz", 未選択}, // Last block is "archive.tar.gz"
		{"Empty String Input", "", 未選択},                            // filepath.Base("") is ".", Split(".") is [".", ""], last is "" -> "未選択" (behavior might vary slightly by OS, but "." or "" are common)
		{"Just a Hyphen", "-", 未選択},                                // filepath.Base("-") is "-", Split("-") is ["", ""], last is "" -> "未選択"
		{"Hyphen at End", "file-S-", 未選択},                          // filepath.Base("file-S-") is "file-S-", Split is ["file", "S", ""], last is "" -> "未選択"
		{"Hyphen at Start and End", "-file-S-", 未選択},               // filepath.Base is "-file-S-", Split is ["", "file", "S", ""], last is "" -> "未選択"

		// --- Directory Path Cases ---
		{"Directory Path - No File", "/path/to/dir/", 未選択}, // filepath.Base is "dir"
		{"Root Directory Path", "/", 未選択},                  // filepath.Base is "/"
		{"Current Directory", ".", 未選択},                    // filepath.Base is "."
		{"Parent Directory", "..", 未選択},                    // filepath.Base is ".."
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := parseOrderType(tc.filePath)
			if actual != tc.expected {
				t.Errorf("parseOrderType(%q): Expected %q, Got %q", tc.filePath, tc.expected, actual)
			}
		})
	}
}
