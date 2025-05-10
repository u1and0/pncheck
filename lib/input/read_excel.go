package input

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		return
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

	// 入力IIの要求年月日とファイル名の要求年月日に矛盾を確認
	s1, err := parseDateSafe(sheet.Header.RequestDate)
	if err != nil {
		err = fmt.Errorf("時間型パースエラー: %s, %w", sheet.Header.RequestDate, err)
		return
	}
	d1, err := time.Parse(dateLayout, s1)
	if err != nil {
		err = fmt.Errorf("時間型パースエラー: %s, %w", sheet.Header.RequestDate, err)
		return
	}
	d2, err := parseFilenameDate(filePath)
	if err != nil {
		err = fmt.Errorf("時間型パースエラー: %s, %w", filePath, err)
		return
	}
	if d1 != d2 {
		err = errors.New("入力IIの要求年月日とファイル名の要求年月日に矛盾があります")
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

	newFilePath := modifyFilePath(filePath)
	if err := f.SaveAs(newFilePath); err != nil {
		return fmt.Errorf("ファイル書き込みエラー: %w\n", err)
	}
	return fmt.Errorf("入力Iをアクティブにして%sへ新しく保存しました。",
		newFilePath)
}

// modifyFilePath : ファイル名の接頭にpncheck_をつける
// filePathにはディレクトリが含まれており、
// ディレクトリとファイル名を分離して、
// ファイル名のprefixに"pncheck_"をつけて新しいファイルパスとする
func modifyFilePath(filePath string) string {
	var (
		// ディレクトリとファイル名を分離
		dir      = filepath.Dir(filePath)
		fileName = filepath.Base(filePath)
	)
	const prefix = "pncheck_"
	// ファイル名のprefixに"pncheck_"をつける
	newFileName := prefix + fileName

	// 新しいファイルパスを生成
	return filepath.Join(dir, newFileName)
}
