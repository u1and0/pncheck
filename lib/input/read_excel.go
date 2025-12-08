/*
inputパッケージでは エクセルファイルの読込みを主に担当します。

- read_excel.go : excelファイルの読み込み関連モジュール

- order.go : 発注区分の決定をサポートします。

- sheet.go : ExcelのデータをPNSearch APIへ送るのに適したJSON型 Sheet構造体へ変換します。
*/
package input

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/xuri/excelize/v2"
)

var sheetsToValidate = []string{
	"入力II",
	"10品目用",
	"30品目用",
	"100品目用",
}

type sheetValidationConfig struct {
	cellRange    string
	cellSum      string
	upperSumCell string // AX7 for print sheets, O7 for InputII
}

// ReadExcelToSheet は指定されたExcelファイルを読み込み、Sheet構造体に変換します。
// Excelのレイアウトは提供された書き込みコードに基づいて定数で定義されたものを仮定しています。
func ReadExcelToSheet(filePath string) (sheet Sheet, err error) {
	if err = validateFile(filePath); err != nil {
		return
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return sheet, fmt.Errorf("ファイルを開けません '%s': %w\n", filePath, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			err = fmt.Errorf("警告: ファイルクローズエラー '%s': %v\n", filePath, err)
		}
		// defer だからfmt.Printf()だけにすべき？
	}()

	// 有効なファイルであることを確認できたら、
	// Sheetを作成して、Header,Orderの読み込み
	sheet = *New(filePath)
	// 発注区分以外のヘッダー情報をExcelファイルから読み込み
	if err = sheet.Header.read(f); err != nil {
		err = fmt.Errorf("入力II読み込みエラー: '%s': %w\n", filePath, err)
		return
	}

	// オーダー情報をExcelファイルから読み込み
	if err = sheet.Orders.read(f); err != nil {
		err = fmt.Errorf("入力I読み込みエラー: '%s': %w\n", filePath, err)
		return
	}

	if len(sheet.Orders) == 0 {
		err = fmt.Errorf("警告: ファイル '%s' のシート '%s' から明細データを読み取れませんでした。\n", filePath, orderSheetName)
		return
	}

	// 各シートの合計値の検証
	if err = validateExcelSums(f, filePath); err != nil {
		return sheet, err
	}

	return
}

// validateExcelSums はExcelシート内の合計値が正しいか検証します。
func validateExcelSums(f *excelize.File, filePath string) error {
	for _, sheetName := range sheetsToValidate {
		if _, err := f.GetSheetIndex(sheetName); err != nil {
			slog.Warn(fmt.Sprintf("シート '%s' が見つかりません。スキップします。", sheetName), slog.String("sheet", sheetName))
			continue
		}

		config, ok := getSheetValidationConfig(sheetName)
		if !ok {
			slog.Warn(fmt.Sprintf("不明なシート名 '%s' です。スキップします。", sheetName), slog.String("sheet", sheetName))
			continue
		}

		sum, err := sumCellRange(f, sheetName, config.cellRange)
		if err != nil {
			return fmt.Errorf("印刷シート '%s' %s の合計計算エラー: %w", sheetName, config.cellRange, err)
		}
		valAX7 := getFloatCellValue(f, sheetName, config.upperSumCell)
		valCellSum := getFloatCellValue(f, sheetName, config.cellSum)
		if sum != valAX7 || sum != valCellSum {
			return fmt.Errorf("Error: Excelファイル '%s' のシート '%s' において、%s の合計 (%.2f) が AX7 (%.2f) または %s (%.2f) と異なります",
				filePath, sheetName, config.cellRange, sum, valAX7, config.cellSum, valCellSum)
		}
	}

	return nil
}

// getSheetValidationConfig はシート名に基づいて検証設定を返します。
func getSheetValidationConfig(sheetName string) (sheetValidationConfig, bool) {
	switch sheetName {
	case "入力II":
		return sheetValidationConfig{cellRange: "O10:O109", cellSum: "O7", upperSumCell: "O7"}, true
	case "10品目用":
		return sheetValidationConfig{cellRange: "AY13:AY22", cellSum: "AY23", upperSumCell: "AX7"}, true
	case "30品目用":
		return sheetValidationConfig{cellRange: "AY13:AY42", cellSum: "AY43", upperSumCell: "AX7"}, true
	case "100品目用":
		return sheetValidationConfig{cellRange: "AY13:AY112", cellSum: "AY113", upperSumCell: "AX7"}, true
	default:
		return sheetValidationConfig{}, false
	}
}

// validateFile : ファイルタイプを検証する
func validateFile(f string) error {
	// 渡されたファイルがディレクトリの場合は無視
	fileInfo, err := os.Stat(f)
	if err != nil {
		return fmt.Errorf("ファイル情報読み込みエラー: %w\n", err)
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("%s はディレクトリです\n", f)
	}
	return nil
}

// ActivateOrderSheet : 入力I以外がアクティブシートだったら
// 入力Iをアクティブにして保存して終了
func ActivateOrderSheet(filePath string) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("ファイルを開けません '%s': %w\n", filePath, err)
	}
	defer f.Close()

	activeSheetIndex := f.GetActiveSheetIndex()
	idx, err := f.GetSheetIndex(orderSheetName)
	if err != nil || idx == -1 {
		return fmt.Errorf("入力Iシートが見つかりません: %w\n", err)
	}

	// 現在のアクティブシートが入力Iだったら何もせずに終了
	if idx == activeSheetIndex {
		return nil
	}

	// 入力I以外がアクティブシートだったら
	// 入力Iをアクティブにして保存して終了
	f.SetActiveSheet(idx)

	if err := f.SaveAs(filePath); err != nil {
		return fmt.Errorf("ファイル書き込みエラー: %w\n", err)
	}
	fmt.Printf("入力Iをアクティブにして%sへ上書き保存しました。", filePath)
	return nil
}
