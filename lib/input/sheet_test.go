package input

import (
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
		Config: Config{Validatable: true, Sortable: true},
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
