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

// TestReadExcelToSheet_Success
// expectedSheet.Header.ProjectID の期待値を修正
func TestReadExcelToSheet_Success(t *testing.T) {
	testDir := "testdata_read"
	testFile := createTestExcelFile(t, testDir, "success_read.xlsx", setValidLayout)

	expectedSheet := Sheet{
		Config: Config{Validatable: true, Sortable: false},
		Header: Header{
			OrderType:   購入,                  // ファイル名から取得される (※要確認セル B2 -> parseFileNameInfo に変更)
			ProjectID:   "1234501",           // ★ 修正: 親番("12345") + 枝番("01") で結合された文字列
			ProjectName: "テストプロジェクト",         // D5
			RequestDate: "2023/10/27",        // D4
			Deadline:    "2023/11/30",        // D2
			FileName:    "success_read.xlsx", // ファイル名
			// UserSection: "開発部",       // UserSection は読み込まなくなったので削除
			Note: "備考欄テスト", // D6
		},
		Orders: Orders{
			// ... (Orders の期待値は変更なし) ...
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

	// 実行前に、テスト用のファイル名が parseFileNameInfo でエラーにならないように調整
	// (parseFileNameInfo はファイル名自体を引数にとるため)
	// expectedFileName := fmt.Sprintf("20231027-%s-%s-%s.xlsx",
	//     expectedSheet.Header.ProjectID[:12], // 製番部分 (12桁想定)
	//     "DUMMY", // 号機はテストデータ設定にないので仮
	//     "K") // OrderTypeが購入になるように
	// testFile = createTestExcelFile(t, testDir, expectedFileName, setValidLayout) // 再生成 or リネーム
	// ↑ parseFileNameInfo が filePath を受け取るので、テストファイル自体の名前を期待値に合わせるか、
	//   parseFileNameInfo の引数を固定値にするなどの工夫が必要

	// ---- parseFileNameInfo を直接呼び出すのではなく、readExcelToSheet内で呼ばれることを考慮 ----
	// readExcelToSheet は filePath を引数にとり、その中で parseFileNameInfo(filePath) を呼ぶ
	// そのため、テスト用のExcelファイル名自体が parseFileNameInfo のフォーマットに合致している必要がある

	// テストファイル名を parseFileNameInfo が成功する形式にする
	// 例: "YYYYMMDD-ProjectID(親)-Serial-OrderType.xlsx"
	// このテストでは OrderType 購入 (K) を期待
	correctFormatFileName := fmt.Sprintf("20231027-%s-TESTSERIAL-K.xlsx", expectedSheet.Header.ProjectID)
	// createTestExcelFile でファイル名を設定し直すか、テスト前にリネームする
	// ここでは createTestExcelFile に渡すファイル名を変更する
	testFile = createTestExcelFile(t, testDir, correctFormatFileName, setValidLayout)
	// 期待値の FileName も合わせる
	expectedSheet.Header.FileName = correctFormatFileName

	actualSheet, err := readExcelToSheet(testFile)
	if err != nil {
		t.Fatalf("予期せぬエラーが発生しました: %v", err)
	}

	// DeepEqual で比較
	if !reflect.DeepEqual(expectedSheet, actualSheet) {
		// ... (詳細比較のロジックは変更なし) ...
		t.Errorf("Sheetが期待値と異なります。")
		if !reflect.DeepEqual(expectedSheet.Config, actualSheet.Config) {
			t.Errorf("  Config:\n    期待値: %+v\n    実際値: %+v", expectedSheet.Config, actualSheet.Config)
		}
		if !reflect.DeepEqual(expectedSheet.Header, actualSheet.Header) {
			t.Errorf("  Header:\n    期待値: %+v\n    実際値: %+v", expectedSheet.Header, actualSheet.Header)
		}
		// ... (Ordersの比較など) ...
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

// expectedSheet.Header.ProjectID の期待値を修正
func TestReadExcelToSheet_EmptySheet(t *testing.T) {
	testDir := "testdata_read"
	// ファイル名を parseFileNameInfo が成功する形式にする
	correctFormatFileName := "20240101-9999900-EMPTY-K.xlsx"

	testFile := createTestExcelFile(t, testDir, correctFormatFileName, func(f *excelize.File) {
		// ヘッダーだけ設定し、明細は空にする
		f.SetCellValue(headerSheetName, projectIDCell, "99999") // 親番
		f.SetCellValue(headerSheetName, projectEdaCell, "00")   // 枝番
		f.SetCellValue(headerSheetName, projectNameCell, "空シートテスト")
		// ... 他のヘッダー項目 ...
		// orderSheetName ("入力Ⅰ") には何も書き込まない
	})

	expectedHeader := Header{
		OrderType:   購入,        // ファイル名から
		ProjectID:   "9999900", // 親番 + 枝番
		ProjectName: "空シートテスト",
		// RequestDate など、設定していないフィールドはゼロ値のまま
		FileName: correctFormatFileName,
	}

	sheet, err := readExcelToSheet(testFile)
	if err != nil {
		// parseFileNameInfo でエラーになる可能性があるためチェック
		t.Fatalf("空の明細シートで予期せぬエラー: %v", err)
	}
	if len(sheet.Orders) != 0 {
		t.Errorf("明細が空のはずが、%d件読み込まれました。", len(sheet.Orders))
	}
	// ヘッダーを比較 (ProjectID と FileName、OrderType のみチェック)
	if sheet.Header.ProjectID != expectedHeader.ProjectID {
		t.Errorf("空の明細シートでもヘッダーは読み込まれるはずです。ProjectID: 期待=%q, 実際=%q", expectedHeader.ProjectID, sheet.Header.ProjectID)
	}
	if sheet.Header.FileName != expectedHeader.FileName {
		t.Errorf("FileNameが期待値と異なります。FileName: 期待=%q, 実際=%q", expectedHeader.FileName, sheet.Header.FileName)
	}
	if sheet.Header.OrderType != expectedHeader.OrderType {
		t.Errorf("OrderTypeが期待値と異なります。OrderType: 期待=%q, 実際=%q", expectedHeader.OrderType, sheet.Header.OrderType)
	}
	t.Log("空の明細シートを正常に処理しました。")
}
