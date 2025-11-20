package input

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
	"pncheck/lib/output"
)

// createTestExcelFile はテスト用のExcelファイルを作成します。
func createTestExcelFile(t *testing.T, dir, filename string, layoutFunc func(f *excelize.File)) string {
	t.Helper()
	filePath := filepath.Join(os.TempDir(), dir, filename) // Use os.TempDir() for isolation
	f := excelize.NewFile()

	// 必要なシートを作成
	_, _ = f.NewSheet(headerSheetName)       // "入力Ⅱ"
	_, _ = f.NewSheet(orderSheetName)        // "入力Ⅰ"
	_, _ = f.NewSheet(printSheetNameDefault) // "10品目用"

	// 不要になったデフォルトシートを削除 (NewFileで作成される "Sheet1")
	// Note: シートが存在しない場合のエラーは無視する
	_ = f.DeleteSheet("Sheet1")

	// レイアウト関数でファイル内容を設定
	if layoutFunc != nil {
		layoutFunc(f)
	}

	// ディレクトリ作成
	if err := os.MkdirAll(filepath.Join(os.TempDir(), dir), 0755); err != nil {
		t.Fatalf("テストディレクトリ作成失敗: %v", err)
	}

	// ファイル保存
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("テストファイル保存失敗 '%s': %v", filePath, err)
	}

	// テスト終了時にファイルを削除
	t.Cleanup(func() {
		// "pncheck_" prefixファイルの削除 (存在する場合)
		_ = os.Remove(output.ModifyFileExt(filePath, ".pncheck.xlsx"))
		// 元のテストファイルの削除
		_ = os.Remove(filePath)
		// ディレクトリが空でなくても削除できるように os.RemoveAll を使用
		_ = os.RemoveAll(filepath.Join(os.TempDir(), dir))
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
	f.SetCellValue(printSheetNameDefault, versionCell, "M-701-04") // AV1: 版番号

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
