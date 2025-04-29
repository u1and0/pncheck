package output

import (
	"encoding/json"
	"fmt"
	"os"

	"pncheck/lib/input"
)

type (
	ErrorRecord struct {
		Message string `json:"msg"`               // 例: "品番が見つかりません"
		Details string `json:"details,omitempty"` // 例: "PN-XXX"
		Key     string `json:"key,omitempty"`     // 例: "品番" or "Pid"
		Index   *int   `json:"index,omitempty"`   // エラーが発生した行番号 (0-based or 1-based? API仕様による)
	}

	PNResponse struct {
		Message string        `json:"msg"`              // ログの概要 (例: "チェック完了", "エラーあり")
		Error   []ErrorRecord `json:"errors,omitempty"` // エラーがあればErrorRecordを追記
		SHA256  string        `json:"sha256,omitempty"` // Sheet構造体から計算したsha256ハッシュ
		Sheet   input.Sheet   `json:"sheet,omitempty"`  // 検証対象のSheet構造体 (オプション)
	}
)

// HandleAPIResponse は APIレスポンスボディ (JSON) を PNResponse 構造体にデコードします。
func HandleAPIResponse(responseBody []byte) (PNResponse, error) {
	var pnResponse PNResponse
	if err := json.Unmarshal(responseBody, &pnResponse); err != nil {
		return PNResponse{}, fmt.Errorf("APIレスポンスJSONのデコードに失敗しました: %w", err)
	}
	return pnResponse, nil
}

// JSONをファイルに書き込む関数
func (data *PNResponse) WriteJSON(filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, jsonData, 0644)
}
