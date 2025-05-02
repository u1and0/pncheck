package output

import (
	"fmt"
	"log"
	"os"
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
