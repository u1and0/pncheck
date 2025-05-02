package input

import (
	"fmt"
	"os"
	"path/filepath"
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
		return
		// defer だからfmt.Printf()だけにすべき？
	}()

	// 有効なファイルであることを確認できたら、
	// Sheetを作成して、Header,Orderの読み込み
	name := filepath.Base(filePath)
	sheet = *New(name)
	// 発注区分をファイル名から分類
	sheet.Header.OrderType = parseOrderType(filePath)
	// 発注区分以外のヘッダー情報をExcelファイルから読み込み
	if err = sheet.Header.read(f); err != nil {
		return
	}
	// オーダー情報をExcelファイルから読み込み
	if err = sheet.Orders.read(f); err != nil {
		return
	}

	if len(sheet.Orders) == 0 {
		err = fmt.Errorf("警告: ファイル '%s' のシート '%s' から明細データを読み取れませんでした。\n", filePath, orderSheetName)
		return
	}

	return
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
