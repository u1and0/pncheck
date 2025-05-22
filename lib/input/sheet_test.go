package input

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestNew(t *testing.T) {
	filePath := "20220101-12345678-TBD-K.xlsx"
	actual := New(filePath)
	expected := Sheet{
		Config: Config{true, true, true},
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
		},
	}

	f, err := excelize.OpenFile(testFile)
	if err != nil {
		t.Errorf("テスト用Excelファイルが開けません\n")
	}
	defer f.Close()

	actual := *New(testFile)
	actual.Header.read(f)
	if expected.Header != actual.Header {
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
	if expected.Orders[0] != actual.Orders[0] {
		t.Errorf("got %#v, want: %#v", actual.Orders[0], &expected.Orders[0])
	}
}
