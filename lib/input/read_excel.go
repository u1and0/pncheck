package input

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// --- 定数定義 (Excelレイアウト - requestパッケージの書き込みコードに基づく) ---

const (
	headerSheetName = "入力Ⅱ" // ヘッダー情報が主に書かれているシート名
	orderSheetName  = "入力Ⅰ" // 明細情報が書かれているシート名 (requestパッケージの定数を使用)

	// --- Header セル位置 (入力Ⅱ) ---
	projectIDCell   = "D1" // 製番 (親番)
	projectEdaCell  = "F1" // 製番 (枝番) - 親番 + 枝番 => 製番　とする
	deadlineHCell   = "D2" // 製番納期
	requestDateCell = "D4" // 要求年月日
	projectNameCell = "D5" // 製番名称
	noteCell        = "D6" // 備考
	// userSectionCell = "P5" // 要求元 (※要確認: 印刷シートから転記されている想定)
	// orderTypeCell   = "B2" // 発注区分 (※要確認: 書き込みコードに該当なし、テンプレート依存の可能性大)

	// --- Order セル位置 (入力Ⅰ) ---
	ordersStartRow = 2   // 明細行が始まる行
	colLv          = "A" // Lv列
	colPid         = "E" // 品番列
	colName        = "F" // 品名列
	colType        = "G" // 型式列
	colQuantity    = "I" // 数量列
	colDeadlineO   = "J" // 要望納期列
	colKenku       = "K" // 検区列
	colDevice      = "M" // 装置名列
	colSerial      = "N" // 号機列
	colMaker       = "O" // メーカ列
	// colCompositionQty = "Y" // 構成数量 (固定値1のため読み込み不要)
	colUnit           = "BE" // 単位列
	colVendor         = "BF" // 要望先列
	colUnitPrice      = "BG" // 予定単価列
	maxEmptyRowsCheck = 5    // 連続で何行空行なら明細終了とみなすか
)

// ReadExcelToSheet は指定されたExcelファイルを読み込み、Sheet構造体に変換します。
// Excelのレイアウトは提供された書き込みコードに基づいて定数で定義されたものを仮定しています。
func ReadExcelToSheet(filePath string) (Sheet, error) {
	var sheet Sheet

	// 渡されたファイルがディレクトリの場合は無視
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return sheet, fmt.Errorf("ファイル情報読み込みエラー: %w\n", err)
	}

	if fileInfo.IsDir() {
		return sheet, fmt.Errorf("%s はディレクトリです\n", filePath)
	}

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return sheet, fmt.Errorf("ファイルを開けません '%s': %w\n", filePath, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("警告: ファイルクローズエラー '%s': %v\n", filePath, err)
		}
	}()

	// --- Configを設定 (現在は固定値) ---
	sheet.Config = Config{
		Validatable: true, // とりあえずtrueに設定
		Sortable:    true, // pncheckを使う段階ではどちらでも良い
	}

	// --- ヘッダー情報を読み込む (主に headerSheetName = "入力Ⅱ" から) ---
	sheet.Header.FileName = filepath.Base(filePath)

	// 発注区分
	orderType := parseOrderType(filePath)
	sheet.Header.OrderType = orderType

	// 製番 (親番のみ読み取り)
	parentID := getCellValue(f, headerSheetName, projectIDCell)
	edaID := getCellValue(f, headerSheetName, projectEdaCell)
	sheet.Header.ProjectID = parentID + edaID
	// 製番枝番は読み込まない (必要なら sheet.Header にフィールド追加し、projectEdaCell から読み込む)
	// sheet.Header.ProjectEda = getCellValue(f, headerSheetName, projectEdaCell)

	sheet.Header.ProjectName = getCellValue(f, headerSheetName, projectNameCell)
	sheet.Header.RequestDate = getCellValue(f, headerSheetName, requestDateCell)

	// 製番納期 (デフォルト値 "―" を考慮)
	deadlineVal := getCellValue(f, headerSheetName, deadlineHCell)
	if deadlineVal == "―" {
		sheet.Header.Deadline = "" // デフォルト値なら空文字にする（または要件に合わせて "―" のまま）
	} else {
		sheet.Header.Deadline = deadlineVal
	}

	// 要求元 (※要確認セル)
	// sheet.Header.UserSection = getCellValue(f, headerSheetName, userSectionCell)
	sheet.Header.Note = getCellValue(f, headerSheetName, noteCell)

	// --- 明細行 (Orders) を読み込む (orderSheetName = "入力Ⅰ" から) ---
	sheet.Orders = make(Orders, 0)
	emptyRowCount := 0
	for r := ordersStartRow; ; r++ {
		// 1行分のデータを読み込む (主要な列が空かチェック - 品番, 品名, 数量)
		rowPid := getCellValue(f, orderSheetName, colPid+strconv.Itoa(r))
		rowName := getCellValue(f, orderSheetName, colName+strconv.Itoa(r))
		rowQuantityStr := getCellValue(f, orderSheetName, colQuantity+strconv.Itoa(r))

		// 品番、品名、数量がすべて空なら空行とみなす
		if rowPid == "" && rowName == "" && rowQuantityStr == "" {
			emptyRowCount++
			if emptyRowCount >= maxEmptyRowsCheck {
				break // 連続空行が閾値を超えたら終了
			}
			continue // 空行なら次の行へ
		}
		emptyRowCount = 0 // データがあればカウンタリセット

		// --- 1行分のデータをOrder構造体に変換 ---
		var order Order
		lvStr := getCellValue(f, orderSheetName, colLv+strconv.Itoa(r))
		order.Lv, err = strconv.Atoi(strings.TrimSpace(lvStr))
		if err != nil && lvStr != "" {
			return sheet, fmt.Errorf("明細(%s) %d行目: Lv(%s)が数値ではありません: %w", orderSheetName, r, colLv, err)
		}

		order.Pid = rowPid
		order.Name = rowName
		order.Type = getCellValue(f, orderSheetName, colType+strconv.Itoa(r))

		// 数量をパース
		order.Quantity, err = strconv.ParseFloat(strings.TrimSpace(rowQuantityStr), 64)
		if err != nil && rowQuantityStr != "" {
			return sheet, fmt.Errorf("明細(%s) %d行目: 数量(%s)が数値ではありません: %w", orderSheetName, r, colQuantity, err)
		}

		order.Unit = getCellValue(f, orderSheetName, colUnit+strconv.Itoa(r))
		order.Deadline = getCellValue(f, orderSheetName, colDeadlineO+strconv.Itoa(r))
		order.Kenku = getCellValue(f, orderSheetName, colKenku+strconv.Itoa(r))
		order.Device = getCellValue(f, orderSheetName, colDevice+strconv.Itoa(r))
		order.Serial = getCellValue(f, orderSheetName, colSerial+strconv.Itoa(r))
		order.Maker = getCellValue(f, orderSheetName, colMaker+strconv.Itoa(r))
		order.Vendor = getCellValue(f, orderSheetName, colVendor+strconv.Itoa(r))

		// 予定単価をパース
		unitPriceStr := getCellValue(f, orderSheetName, colUnitPrice+strconv.Itoa(r))
		order.UnitPrice, err = strconv.ParseFloat(strings.TrimSpace(unitPriceStr), 64)
		if err != nil && unitPriceStr != "" {
			order.UnitPrice = 0 // エラーの場合は0にする（要件次第）
			// return sheet, fmt.Errorf("明細(%s) %d行目: 予定単価(%s)が数値ではありません: %w", orderSheetName, r, colUnitPrice, err)
		}

		// 読み取ったOrderをスライスに追加
		sheet.Orders = append(sheet.Orders, order)
	}

	if len(sheet.Orders) == 0 {
		fmt.Printf("警告: ファイル '%s' のシート '%s' から明細データを読み取れませんでした。\n", filePath, orderSheetName)
	}

	return sheet, nil
}

// getCellValue は指定されたセルから値を取得します。エラー時は空文字を返します。
func getCellValue(f *excelize.File, sheetName, axis string) string {
	val, err := f.GetCellValue(sheetName, axis)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val)
}

// FilenameWithoutExt : ファイルパスを渡して
// 拡張子なしのファイル名を返す
// ディレクトリの場合、Base名をそのまま返す
func FilenameWithoutExt(filePath string) string {
	if filePath == "" {
		return ""
	}
	base := filepath.Base(filePath)
	ext := filepath.Ext(filePath)
	return strings.TrimSuffix(base, ext)
}
