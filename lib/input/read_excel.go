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
	const (
		cellInput2Row = "O10:O109"
		cellInput2Sum = "O7"
	)

	// 入力IIの検証
	sumInputII, err := sumCellRange(f, headerSheetName, cellInput2Row)
	if err != nil {
		return fmt.Errorf("入力IIシート 合計計算エラー: %w", err)
	}
	valO7InputII := getFloatCellValue(f, headerSheetName, cellInput2Sum)
	if sumInputII != valO7InputII {
		return fmt.Errorf("Error: Excelファイル '%s' のシート '%s' において、金額の合計 (%.2f) が 計算した合計値 (%.2f) と異なります",
			filePath, headerSheetName, sumInputII, valO7InputII)
	}

	// 印刷シート1の検証
	printSheet1Name := getPrintSheet(f) // 動的にシート名を取得
	sumPrint1, err := sumCellRange(f, printSheet1Name, "AY13:AY22")
	if err != nil {
		return fmt.Errorf("印刷シート1 '%s' AY13:AY22 の合計計算エラー: %w", printSheet1Name, err)
	}
	valAX7Print1 := getFloatCellValue(f, printSheet1Name, "AX7")
	valAY23Print1 := getFloatCellValue(f, printSheet1Name, "AY23")
	if sumPrint1 != valAX7Print1 || sumPrint1 != valAY23Print1 {
		return fmt.Errorf("Error: Excelファイル '%s' のシート '%s' において、AY13:AY22 の合計 (%.2f) が AX7 (%.2f) または AY23 (%.2f) と異なります",
			filePath, printSheet1Name, sumPrint1, valAX7Print1, valAY23Print1)
	}

	// 印刷シート2の検証
	printSheet2Name := "印刷シート2"
	if _, err := f.GetSheetIndex(printSheet2Name); err != nil {
		slog.Warn("印刷シート2が見つかりません。スキップします。", slog.String("sheet", printSheet2Name))
	} else {
		sumPrint2, err := sumCellRange(f, printSheet2Name, "AY13:AY42")
		if err != nil {
			return fmt.Errorf("印刷シート2 '%s' AY13:AY42 の合計計算エラー: %w", printSheet2Name, err)
		}
		valAX7Print2 := getFloatCellValue(f, printSheet2Name, "AX7")
		valAY43Print2 := getFloatCellValue(f, printSheet2Name, "AY43")
		if sumPrint2 != valAX7Print2 || sumPrint2 != valAY43Print2 {
			return fmt.Errorf("Error: Excelファイル '%s' のシート '%s' において、AY13:AY42 の合計 (%.2f) が AX7 (%.2f) または AY43 (%.2f) と異なります",
				filePath, printSheet2Name, sumPrint2, valAX7Print2, valAY43Print2)
		}
	}

	// 印刷シート3の検証
	printSheet3Name := "印刷シート3"
	if _, err := f.GetSheetIndex(printSheet3Name); err != nil {
		slog.Warn("印刷シート3が見つかりません。スキップします。", slog.String("sheet", printSheet3Name))
	} else {
		sumPrint3, err := sumCellRange(f, printSheet3Name, "AY13:AY112")
		if err != nil {
			return fmt.Errorf("印刷シート3 '%s' AY13:AY112 の合計計算エラー: %w", printSheet3Name, err)
		}
		valAX7Print3 := getFloatCellValue(f, printSheet3Name, "AX7")
		valAY113Print3 := getFloatCellValue(f, printSheet3Name, "AY113")
		if sumPrint3 != valAX7Print3 || sumPrint3 != valAY113Print3 {
			return fmt.Errorf("Error: Excelファイル '%s' のシート '%s' において、AY13:AY112 の合計 (%.2f) が AX7 (%.2f) または AY113 (%.2f) と異なります",
				filePath, printSheet3Name, sumPrint3, valAX7Print3, valAY113Print3)
		}
	}

	return nil
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
