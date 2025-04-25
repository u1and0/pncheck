package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// サーバーからのエラー出力のうち、このコマンドで使用するもの
type ErrorOutput struct {
	Filename string        `json:"filename"`
	Msg      string        `json:"msg"`
	Errors   []ErrorRecord `json:"errors,omitempty"`
}

// writeErrorFile は集約されたエラー結果を指定されたファイルにJSON形式で出力します。
// エラーがない場合はファイルを作成しません。
// 出力形式: エラーがあった FileProcessResult のスライスをインデント付きJSONで出力
func (eo *ErrorOutput) WriteErrorFile(path string) error {
	// ファイルを開く (なければ作成、あれば上書き)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' の作成に失敗しました: %w", path, err)
	}
	defer file.Close()

	b, err := json.MarshalIndent(&eo, "", "  ")
	_, err = file.Write(b)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' へのJSONデータ書き込みに失敗しました: %w", path, err)
	}
	return nil
}
