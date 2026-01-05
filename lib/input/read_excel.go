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
	"strings"

	"github.com/xuri/excelize/v2"
)

// 合計値を確認するシート名
var sheetsToValidate = []string{
	"入力Ⅱ",
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

	opts := excelize.Options{RawCellValue: true}
	f, err := excelize.OpenFile(filePath, opts)
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

	return
}

// ValidateExcelSums はExcelシート内の合計値が正しいか検証します。
func ValidateExcelSums(filePath string) error {
	opts := excelize.Options{RawCellValue: true}
	f, err := excelize.OpenFile(filePath, opts)
	if err != nil {
		return fmt.Errorf("ファイルを開けません '%s': %w\n", filePath, err)
	}
	defer f.Close()

	for _, sheetName := range sheetsToValidate {
		i, err := f.GetSheetIndex(sheetName)
		if err != nil || i < 0 {
			slog.Warn(fmt.Sprintf("シート '%s' が見つかりません。スキップします。", sheetName), slog.String("sheet", sheetName))
			continue
		}

		// レンジの合計値算出
		config, err := getSheetValidationConfig(f, sheetName)
		if err != nil {
			return fmt.Errorf("%sシートの合計計算設定エラー: %w", sheetName, err)
		}
		sum, err := sumCellRange(f, sheetName, config.cellRange)
		if err != nil {
			return fmt.Errorf("%sシートの合計計算エラー: %w", sheetName, err)
		}

		// レンジの合計と上下それぞれの合計値が等しくなければエラーを返す
		err = fmt.Errorf(
			"%sシートにおいて、%s の合計が正しく計算できていません",
			sheetName, config.cellRange,
		)
		valUpperSumCell := getFloatCellValue(f, sheetName, config.upperSumCell)
		if sum != valUpperSumCell {
			return err
		}
		valCellSum := getFloatCellValue(f, sheetName, config.cellSum)
		if sum != valCellSum {
			return err
		}
	}

	return nil
}

// getSheetValidationConfig はシート名に基づいて検証設定を返します。
// 入力II シートの場合は固定値を返す。
//
// それ以外のシートでは、AY*** 数値が可変。
// AU列をループで回していって、
// "合計"という文字列が12行目以降に出てきた行を合計値の行とする。
//
// 例えばAU111 に"合計"という文字列がある場合、
// cellSum=AY111 として合計値を計算する
func getSheetValidationConfig(f *excelize.File, sheetName string) (sheetValidationConfig, error) {
	if sheetName == "入力Ⅱ" {
		config := sheetValidationConfig{cellRange: "O10:O109", cellSum: "O7", upperSumCell: "O7"}
		return config, nil
	}

	var sumRow = 13 // シート下側の合計値の行数
	// AU *** に"合計"という文字列のサーチ
	for {
		ax := fmt.Sprintf("AU%d", sumRow)
		s, err := f.GetCellValue(sheetName, ax)
		if err != nil {
			return sheetValidationConfig{}, err
		}
		if strings.TrimSpace(s) == "合計" {
			break
		}
		sumRow++
	}
	config := sheetValidationConfig{
		cellRange:    fmt.Sprintf("AY13:AY%d", sumRow-1),
		cellSum:      fmt.Sprintf("AY%d", sumRow),
		upperSumCell: "AX7",
	}
	return config, nil
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
