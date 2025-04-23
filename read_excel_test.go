package main

import (
	"fmt" // fmtを追加
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// --- テストヘルパー: テスト用Excelファイル作成 ---

// createTestExcelFile はテスト用のExcelファイルを作成します。
func createTestExcelFile(t *testing.T, dir, filename string, layoutFunc func(f *excelize.File)) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	f := excelize.NewFile()

	// デフォルトシート名を削除 (テストで必要なシートを作成するため)
	// f.DeleteSheet("Sheet1") // excelizeのバージョンによっては不要/エラーになる場合あり

	// 必要なシートを作成
	_, _ = f.NewSheet(headerSheetName) // "入力Ⅱ"
	_, _ = f.NewSheet(orderSheetName)  // "入力Ⅰ"

	// 不要になったデフォルトシートを削除 (NewFileで作成される "Sheet1")
	// Note: シートが存在しない場合のエラーは無視する
	_ = f.DeleteSheet("Sheet1")

	// レイアウト関数でファイル内容を設定
	if layoutFunc != nil {
		layoutFunc(f)
	} else {
		setValidLayout(f) // デフォルトの正常系レイアウトを設定
	}

	// ディレクトリ作成
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("テストディレクトリ作成失敗: %v", err)
	}

	// ファイル保存
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("テストファイル保存失敗 '%s': %v", filePath, err)
	}

	// テスト終了時にファイルを削除
	t.Cleanup(func() {
		os.Remove(filePath)
		os.Remove(dir) // ディレクトリが空なら削除
	})
	return filePath
}

// setValidLayout は正常系のExcelレイアウトを設定します。
func setValidLayout(f *excelize.File) {
	// --- Header (入力Ⅱ) ---
	f.SetCellValue(headerSheetName, projectIDCell, " 12345 ")      // D1: 製番(親)
	f.SetCellValue(headerSheetName, projectEdaCell, "01")          // F1: 製番(枝) - 読み込み対象外
	f.SetCellValue(headerSheetName, deadlineHCell, "2023/11/30")   // D2: 製番納期
	f.SetCellValue(headerSheetName, requestDateCell, "2023/10/27") // D4: 要求年月日
	f.SetCellValue(headerSheetName, projectNameCell, "テストプロジェクト")  // D5: 製番名称
	f.SetCellValue(headerSheetName, noteCell, "備考欄テスト")            // D6: 備考

	// --- Orders Header (入力Ⅰ - 見出し行、読み込み対象外だが参考として) ---
	f.SetCellValue(orderSheetName, colLv+"1", "Lv")
	f.SetCellValue(orderSheetName, colPid+"1", "品番")
	f.SetCellValue(orderSheetName, colName+"1", "品名")
	f.SetCellValue(orderSheetName, colQuantity+"1", "数量")
	// ... 他の見出し

	// --- Orders Data (入力Ⅰ - Row 2) ---
	f.SetCellValue(orderSheetName, colLv+"2", " 1 ")
	f.SetCellValue(orderSheetName, colPid+"2", "PN-001")
	f.SetCellValue(orderSheetName, colName+"2", "部品A")
	f.SetCellValue(orderSheetName, colType+"2", "TypeX")
	f.SetCellValue(orderSheetName, colQuantity+"2", " 10.5 ")
	f.SetCellValue(orderSheetName, colUnit+"2", "個")
	f.SetCellValue(orderSheetName, colDeadlineO+"2", "2023/11/15")
	f.SetCellValue(orderSheetName, colKenku+"2", "受入")
	f.SetCellValue(orderSheetName, colDevice+"2", "装置1")
	f.SetCellValue(orderSheetName, colSerial+"2", "S001")
	f.SetCellValue(orderSheetName, colMaker+"2", "MakerX")
	f.SetCellValue(orderSheetName, colVendor+"2", "VendorY")
	f.SetCellValue(orderSheetName, colUnitPrice+"2", " 100.50 ")

	// --- Orders Data (入力Ⅰ - Row 3) ---
	f.SetCellValue(orderSheetName, colLv+"3", "2")
	f.SetCellValue(orderSheetName, colPid+"3", "PN-002")
	f.SetCellValue(orderSheetName, colName+"3", "部品B")
	f.SetCellValue(orderSheetName, colQuantity+"3", "5")
	f.SetCellValue(orderSheetName, colUnit+"3", "Set")
	f.SetCellValue(orderSheetName, colUnitPrice+"3", "2500") // 型式、納期などは空

	// --- 空行 (Row 4) --- スキップされるはず

	// --- Orders Data (入力Ⅰ - Row 5) --- 空行の後
	f.SetCellValue(orderSheetName, colPid+"5", "PN-003")
	f.SetCellValue(orderSheetName, colName+"5", "部品C")
	f.SetCellValue(orderSheetName, colQuantity+"5", "1")
}

// --- テスト関数 ---

func TestReadExcelToSheet_Success(t *testing.T) {
	testDir := "testdata_read" // testdata ディレクトリ名を変更
	testFile := createTestExcelFile(t, testDir, "success_read-K.xlsx", setValidLayout)

	expectedSheet := Sheet{
		Config: Config{Validatable: true, Sortable: false},
		Header: Header{
			OrderType:   "購入",         // ※要確認セル B2
			ProjectID:   12345,        // D1
			ProjectName: "テストプロジェクト",  // D5
			RequestDate: "2023/10/27", // D4
			Deadline:    "2023/11/30", // D2
			FileName:    "success_read-K.xlsx",
			// UserSection: "開発部",    // ※要確認セル P5
			Note: "備考欄テスト", // D6
		},
		Orders: Orders{
			{ // Row 2
				Lv:        1,
				Pid:       "PN-001",
				Name:      "部品A",
				Type:      "TypeX",
				Quantity:  10.5,
				Unit:      "個",
				Deadline:  "2023/11/15",
				Kenku:     "受入",
				Device:    "装置1",
				Serial:    "S001",
				Maker:     "MakerX",
				Vendor:    "VendorY",
				UnitPrice: 100.50,
			},
			{ // Row 3
				Lv:        2,
				Pid:       "PN-002",
				Name:      "部品B",
				Type:      "", // 空
				Quantity:  5,
				Unit:      "Set",
				Deadline:  "", // 空
				Kenku:     "", // 空
				Device:    "", // 空
				Serial:    "", // 空
				Maker:     "", // 空
				Vendor:    "", // 空
				UnitPrice: 2500,
			},
			{ // Row 5 (After empty row)
				Lv:        0, // 空
				Pid:       "PN-003",
				Name:      "部品C",
				Type:      "", // 空
				Quantity:  1,
				Unit:      "", // 空
				Deadline:  "", // 空
				Kenku:     "", // 空
				Device:    "", // 空
				Serial:    "", // 空
				Maker:     "", // 空
				Vendor:    "", // 空
				UnitPrice: 0,  // 空
			},
		},
	}

	actualSheet, err := readExcelToSheet(testFile)
	if err != nil {
		t.Fatalf("予期せぬエラーが発生しました: %v", err)
	}

	// DeepEqual で比較
	if !reflect.DeepEqual(expectedSheet, actualSheet) {
		// どこが違うか分かりやすく表示
		t.Errorf("Sheetが期待値と異なります。")
		if !reflect.DeepEqual(expectedSheet.Config, actualSheet.Config) {
			t.Errorf("  Config:\n    期待値: %+v\n    実際値: %+v", expectedSheet.Config, actualSheet.Config)
		}
		if !reflect.DeepEqual(expectedSheet.Header, actualSheet.Header) {
			t.Errorf("  Header:\n    期待値: %+v\n    実際値: %+v", expectedSheet.Header, actualSheet.Header)
		}
		if len(expectedSheet.Orders) != len(actualSheet.Orders) {
			t.Errorf("  Ordersの件数が異なります: 期待値=%d, 実際値=%d", len(expectedSheet.Orders), len(actualSheet.Orders))
		} else {
			for i := range expectedSheet.Orders {
				if !reflect.DeepEqual(expectedSheet.Orders[i], actualSheet.Orders[i]) {
					t.Errorf("  Orders[%d]:\n    期待値: %+v\n    実際値: %+v", i, expectedSheet.Orders[i], actualSheet.Orders[i])
				}
			}
		}
	}
}

func TestReadExcelToSheet_FileNotFound(t *testing.T) {
	_, err := readExcelToSheet("testdata_read/non_existent_file-K.xlsx")
	if err == nil {
		t.Fatal("ファイルが存在しない場合にエラーが返されませんでした。")
	}
	t.Logf("期待通りファイルオープンエラーを検出: %v", err)
}

func TestReadExcelToSheet_InvalidNumberFormat_Header(t *testing.T) {
	testDir := "testdata_read"
	testFile := createTestExcelFile(t, testDir, "invalid_header_read-K.xlsx", func(f *excelize.File) {
		setValidLayout(f)
		f.SetCellValue(headerSheetName, projectIDCell, "ABCDE") // D1に文字列
	})

	_, err := readExcelToSheet(testFile)
	if err == nil {
		t.Fatal("ヘッダーの数値変換エラーが検出されませんでした。")
	}
	expectedErrMsg := fmt.Sprintf("ヘッダー(%s): 製番(%s)が数値ではありません", headerSheetName, projectIDCell)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("期待されるエラーメッセージが含まれていません。\n期待含む: %s\n実際: %v", expectedErrMsg, err)
	}
	t.Logf("期待通りヘッダー数値エラーを検出: %v", err)
}

func TestReadExcelToSheet_InvalidNumberFormat_Orders(t *testing.T) {
	testDir := "testdata_read"
	testFile := createTestExcelFile(t, testDir, "invalid_orders_read-K.xlsx", func(f *excelize.File) {
		setValidLayout(f)
		f.SetCellValue(orderSheetName, colQuantity+"2", "Not A Number") // 2行目の数量に文字列
	})

	_, err := readExcelToSheet(testFile)
	if err == nil {
		t.Fatal("明細行の数値変換エラーが検出されませんでした。")
	}
	expectedErrMsg := fmt.Sprintf("明細(%s) 2行目: 数量(%s)が数値ではありません", orderSheetName, colQuantity)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("期待されるエラーメッセージが含まれていません。\n期待含む: %s\n実際: %v", expectedErrMsg, err)
	}
	t.Logf("期待通り明細数値エラーを検出: %v", err)
}

func TestReadExcelToSheet_EmptySheet(t *testing.T) {
	testDir := "testdata_read"
	testFile := createTestExcelFile(t, testDir, "empty_sheet_read-K.xlsx", func(f *excelize.File) {
		// ヘッダーだけ設定し、明細は空にする
		f.SetCellValue(headerSheetName, projectIDCell, "99999")     // D1
		f.SetCellValue(headerSheetName, projectNameCell, "空シートテスト") // D5
		// ... 他のヘッダー項目 ...
		// orderSheetName ("入力Ⅰ") には何も書き込まない
	})

	sheet, err := readExcelToSheet(testFile)
	if err != nil {
		t.Fatalf("空の明細シートで予期せぬエラー: %v", err)
	}
	if len(sheet.Orders) != 0 {
		t.Errorf("明細が空のはずが、%d件読み込まれました。", len(sheet.Orders))
	}
	if sheet.Header.ProjectID != 99999 {
		t.Errorf("空の明細シートでもヘッダーは読み込まれるはずです。ProjectID=%d", sheet.Header.ProjectID)
	}
	t.Log("空の明細シートを正常に処理しました。")
}
