package input

import (
	"reflect"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestNew(t *testing.T) {
	filePath := "20220101-12345678-TBD-K.xlsx"
	actual := New(filePath)
	expected := Sheet{
		Config: Config{true, true},
		Header: Header{
			FileName:  "pncheck_" + filePath,
			OrderType: 購入,
		},
	}
	if actual.Config != expected.Config {
		t.Errorf("got %#v, want: %#v", actual.Config, &expected.Config)
	}
	if actual.Header != expected.Header {
		t.Errorf("got %#v, want: %#v", actual.Header, &expected.Header)

	}
}

// setValidLayout は正常系のExcelレイアウトを設定します。
func setValidHeader(f *excelize.File) {
	// --- Header (入力Ⅱ) ---
	f.SetCellValue(headerSheetName, projectIDCell, " 12345 ")      // D1: 製番(親)
	f.SetCellValue(headerSheetName, projectEdaCell, "01")          // F1: 製番(枝) - 読み込み対象外
	f.SetCellValue(headerSheetName, deadlineHCell, "2023/11/30")   // D2: 製番納期
	f.SetCellValue(headerSheetName, requestDateCell, "2023/10/27") // D4: 要求年月日
	f.SetCellValue(headerSheetName, projectNameCell, "テストプロジェクト")  // D5: 製番名称
	f.SetCellValue(headerSheetName, noteCell, "備考欄テスト")            // D6: 備考
}

func TestHeaderRead(t *testing.T) {
	testDir := "testdata_sheet_header_read"
	testFile := createTestExcelFile(t, testDir, "success-001-read-K.xlsx", setValidLayout)

	expected := Sheet{
		Header: Header{
			OrderType:   購入,
			ProjectID:   "1234501",                         // D1 + F1
			ProjectName: "テストプロジェクト",                       // D5
			RequestDate: "2023/10/27",                      // D4
			Deadline:    "2023/11/30",                      // D2
			FileName:    "pncheck_success-001-read-K.xlsx", // ファイル名 ダミーのpncheck_ prefixがつく
			Note:        "備考欄テスト",                          // D6
			Version:     "M-701-04",                        // AV1
		},
	}

	f, err := excelize.OpenFile(testFile)
	if err != nil {
		t.Errorf("テスト用Excelファイルが開けません\n")
	}
	defer f.Close()

	actual := *New(testFile)
	actual.Header.read(f)
	if !reflect.DeepEqual(actual.Header, expected.Header) {
		t.Errorf("got %#v, want: %#v", actual.Header, &expected.Header)
	}
}

func TestOrderRead(t *testing.T) {
	testDir := "testdata_sheet_order_read"
	testFile := createTestExcelFile(t, testDir, "success-001-read-K.xlsx", setValidLayout)

	expected := Sheet{
		Orders: Orders{
			{ // Row 2
				Lv: 1, Pid: "PN-001", Name: "部品A", Type: "TypeX",
				Quantity: 10.5, Unit: "個", Deadline: "2023/11/15", Kenku: "受入",
				Device: "装置1", Serial: "S001", Maker: "MakerX", Vendor: "VendorY", UnitPrice: 100.50,
			},
			{ // Row 3
				Lv: 2, Pid: "PN-002", Name: "部品B", Type: "",
				Quantity: 5, Unit: "Set", Deadline: "", Kenku: "",
				Device: "", Serial: "", Maker: "", Vendor: "", UnitPrice: 2500,
			},
			{ // Row 5
				Lv: 0, Pid: "PN-003", Name: "部品C", Type: "",
				Quantity: 1, Unit: "", Deadline: "", Kenku: "",
				Device: "", Serial: "", Maker: "", Vendor: "", UnitPrice: 0,
			},
		},
	}

	f, err := excelize.OpenFile(testFile)
	if err != nil {
		t.Errorf("テスト用Excelファイルが開けません\n")
	}
	defer f.Close()

	actual := *New(testFile)
	actual.Orders.read(f)
	if len(actual.Orders) == 0 {
		t.Errorf("Ordersの値がありません len == 0\n")
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
	actual := getLastRemarkValue(f)
	expected := "67890"
	if actual != expected {
		t.Errorf("getLastRemarkValue() = %q, want %q", actual, expected)
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
