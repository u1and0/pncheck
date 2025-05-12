package output

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteErrorToJSON writes error response to a JSON file
func WriteErrorToJSON(jsonPath string, body []byte) error {
	f, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' の作成に失敗しました: %w", jsonPath, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// closeエラーをログに記録するか、既存のエラーに追加する
			// 通常、書き込みエラーの方が重要なので、closeエラーは単にログに記録する
			log.Printf("エラーファイル '%s' のクローズに失敗しました: %v", jsonPath, err)
		}
	}()

	if _, err := f.Write(body); err != nil {
		return fmt.Errorf("エラーファイル '%s' へのJSONデータ書き込みに失敗しました: %w", jsonPath, err)
	}

	return nil
}

// LogFatalError : エラーの内容をエラーログファイルに追記する
func LogFatalError(f string, msg string) error {
	// O_APPEND: ファイルの末尾に書き込む
	// O_CREATE: ファイルが存在しない場合は作成する
	// O_WRONLY: 書き込み専用で開く
	file, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	now := time.Now().Format("2006/01/02 15:04:05.000")
	msg = fmt.Sprintf("%s: %s\n", now, msg)
	_, err = file.WriteString(msg)
	return err
}

// ErrorRecord : pncheck固有のエラーをJSONファイルに書き込むための構造体
type ErrorRecord struct {
	Filename string `json:"ファイル名"`
	Error    string `json:"エラー"`
}

// WriteFatal : pncheck固有のエラーがあったら 標準エラーに出力した後、
// ファイル名ごとのJSONに書き込む
// 引数のerrがnilの場合は何もしないで終了
//
// @throws JSONパースエラー
// @throws エラーファイル '%s' の作成に失敗しました
// @throws エラーファイル '%s' へのJSONデータ書き込みに失敗しました
func WriteFatal(filePath string, err error) error {
	if err == nil {
		return nil
	}
	// エラーをJSONとしてパース
	errRecord := ErrorRecord{filepath.Base(filePath), err.Error()}
	errJSON, err := json.MarshalIndent(errRecord, "", "  ")
	if err != nil {
		return fmt.Errorf("JSONパースエラー: %w", err)
	}

	// JSON型エラーの表示
	// jqでハイライトして見たいので標準出力へ
	fmt.Println(string(errJSON))

	// JSONファイルへ書き込み
	// pncheck実行ディレクトリにJSONを配置する
	jsonFilename := WithoutFileExt(filePath) + ".json"
	return WriteErrorToJSON(jsonFilename, errJSON)
}

// ModifyFileExt
// filePathにはディレクトリが含まれており、
// ディレクトリとファイル名を分離して、
// ファイルの拡張子をセットし直す
func ModifyFileExt(filePath, newExt string) string {
	var (
		// ディレクトリとファイル名と拡張子を分離
		dir      = filepath.Dir(filePath)
		fileBase = WithoutFileExt(filePath)
	)
	return filepath.Join(dir, fileBase+newExt)
}

// WithoutFileExt : ディレクトリパスと拡張子を取り除く
func WithoutFileExt(filePath string) string {
	var (
		// ディレクトリとファイル名と拡張子を分離
		fileName = filepath.Base(filePath)
		ext      = filepath.Ext(filePath)
	)
	return strings.TrimSuffix(fileName, ext)

}
