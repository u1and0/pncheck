package input

import (
	"reflect"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestNew(t *testing.T) {
	filepath := "20220101-12345678-TBD-K.xlsx"
	expected := Sheet{
		Config{true, true},
		Header{
			FileName:  "20220101-12345678-TBD-K_pncheck.xlsx",
			OrderType: 購入,
			Serial:    "TBD",
		},
		Orders{},
	}

	actual := *New(filepath)
	if len(actual.Orders) != 0 {
		t.Errorf("Ordersの値は0のはず: %d\n", len(actual.Orders))
	}
	if !reflect.DeepEqual(actual.Orders, expected.Orders) {
		t.Errorf("got %#v, want: %#v", actual.Orders, expected.Orders)
	}
}

// TestGetLastRemarkValue tests the getLastRemarkValue function
func TestGetLastRemarkValue(t *testing.T) {
	testDir := "testdata_sheet_remark"
	testFile := createTestExcelFile(t, testDir, "remark_test.xlsx", func(f *excelize.File) {
		// Set up test data for the order sheet
		// Row 2
		f.SetCellValue(orderSheetName, colPid+"2", "PN-001")
		f.SetCellValue(orderSheetName, colName+"2", "部品A")
		f.SetCellValue(orderSheetName, colQuantity+"2", "10")
		f.SetCellValue(orderSheetName, colMisc+"2", "出庫指示番号: 12345による")

		// Row 3
		f.SetCellValue(orderSheetName, colPid+"3", "PN-002")
		f.SetCellValue(orderSheetName, colName+"3", "部品B")
		f.SetCellValue(orderSheetName, colQuantity+"3", "5")
		f.SetCellValue(orderSheetName, colMisc+"3", "出庫指示番号: 67890による")

		// Row 4 (empty row to mark end)
		f.SetCellValue(orderSheetName, colPid+"4", "")
		f.SetCellValue(orderSheetName, colName+"4", "")
		f.SetCellValue(orderSheetName, colQuantity+"4", "")
	})

	f, err := excelize.OpenFile(testFile)
	if err != nil {
		t.Errorf("テスト用Excelファイルが開けません\n")
	}
	defer f.Close()

	// Test that we get the last remark value (67890 from row 3)
	actua := getLastRemarkValue(f)
	expected := "67890"
	if actua != expected {
		t.Errorf("getLastRemarkValue() = %q, want %q", actua, expected)
	}
}

// TestCheckOrderItemsSortOrder は CheckOrderItemsSortOrder 関数のテストを行います。
func TestCheckOrderItemsSortOrder(t *testing.T) {
	// テストケースを定義
	tests := []struct {
		name        string // テストケースの名前
		sheet       Sheet  // 入力となるSheetデータ
		expectError bool   // エラーが期待されるか (true: 期待する, false: 期待しない)
		// expectedErrorMsg string // エラーメッセージの詳細をチェックする場合に使う (今回はシンプルにエラーの有無のみ)
	}{
		{
			name:        "Empty Orders - Should Pass",
			sheet:       Sheet{Orders: Orders{}},
			expectError: false,
		},
		{
			name:        "Single Order - Should Pass",
			sheet:       Sheet{Orders: Orders{{Deadline: "2023-10-26", Pid: "A001"}}},
			expectError: false,
		},
		{
			name: "Correct Sort by Deadline Only - Should Pass",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "C003"}, // Pidは順不同でもOK (Deadlineが違うから)
				{Deadline: "2023-10-27", Pid: "A001"},
				{Deadline: "2023-10-28", Pid: "B002"},
			}},
			expectError: false,
		},
		{
			name: "Correct Sort by Deadline and Pid - Should Pass",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "A001"},
				{Deadline: "2023-10-26", Pid: "B002"},
				{Deadline: "2023-10-27", Pid: "C003"},
				{Deadline: "2023-10-27", Pid: "D004"},
				{Deadline: "2023-10-28", Pid: "A001"},
			}},
			expectError: false,
		},
		{
			name: "Incorrect Sort - Deadline Out of Order - Should Fail",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-27", Pid: "C003"},
				{Deadline: "2023-10-26", Pid: "A001"}, // 2023-10-27 の後に 2023-10-26
			}},
			expectError: true,
		},
		{
			name: "Incorrect Sort - Pid Out of Order for Same Deadline - Should Fail",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "B002"},
				{Deadline: "2023-10-26", Pid: "A001"}, // 同じ納期で B002 の後に A001
			}},
			expectError: true,
		},
		{
			name: "Incorrect Sort - Pid Out of Order for Same Deadline (Multiple Items) - Should Fail",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "A001"}, // OK
				{Deadline: "2023-10-27", Pid: "D004"}, // OK (Deadlineが違う)
				{Deadline: "2023-10-27", Pid: "C003"}, // Fail (同じ納期で D004の後に C003)
				{Deadline: "2023-10-28", Pid: "A001"}, // この行には到達しないはず
			}},
			expectError: true,
		},
		{
			name: "Correct Sort with Duplicate Items - Should Pass",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "A001"},
				{Deadline: "2023-10-26", Pid: "A001"}, // 同じアイテムでも順序はOK
				{Deadline: "2023-10-27", Pid: "B002"},
			}},
			expectError: false,
		},
	}

	// 各テストケースを実行
	for _, tt := range tests {
		// t.Run を使うと、各テストケースが独立して実行され、結果が見やすくなります
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sheet.CheckOrderItemsSortOrder()

			// エラーが期待されているかチェック
			if tt.expectError {
				// エラーが期待されているのに nil が返ってきた場合
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
				// (オプション) 特定のエラーメッセージをチェックする場合
				// if err != nil && tt.expectedErrorMsg != "" && err.Error() !=tt.expectedErrorMsg {
				//      t.Errorf("Expected error message '%s', but got '%s'", tt.expectedErrorMsg, err.Error())
				// }
			} else {
				// エラーが期待されていないのに nil 以外が返ってきた場合
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
			}
		})
	}
}

func TestNewFileName(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "xlsx file",
			filePath: "testdata/20231027-success-read-K.xlsx",
			expected: "20231027-success-read-K_pncheck.xlsx",
		},
		{
			name:     "no extension",
			filePath: "testdata/no-extension-file",
			expected: "no-extension-file_pncheck",
		},
		{
			name:     "multiple dots in name",
			filePath: "testdata/file.with.dots.xlsx",
			expected: "file.with.dots_pncheck.xlsx",
		},
		{
			name:     "empty string",
			filePath: "",
			expected: "._pncheck",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actua := newFileName(tt.filePath)
			if actua != tt.expected {
				t.Errorf("newFileName(%q) = %q, want %q", tt.filePath, actua, tt.expected)
			}
		})
	}
}

func TestParseSerial(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "valid file name",
			filePath: "20231027-12345678-S001-K.xlsx",
			expected: "S001",
		},
		{
			name:     "short file name length 2",
			filePath: "20231027-12345678.xlsx",
			expected: "",
		},
		{
			name:     "short file name length 3",
			filePath: "20251114-000080010742-TBP.xlsx",
			expected: "TBP",
		},
		{
			name:     "empty file name",
			filePath: "",
			expected: "",
		},
		{
			name:     "file name with different delimiters",
			filePath: "20231027_12345678_S001_K.xlsx",
			expected: "", // ハイフン区切りではないため
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actua := parseSerial(tt.filePath)
			if actua != tt.expected {
				t.Errorf("parseSerial(%q) = %q, want %q", tt.filePath, actua, tt.expected)
			}
		})
	}
}
