package api

import (
	"encoding/json"
	"errors"
	"fmt"

	"pncheck/lib/input"
)

// APIレスポンス全体の構造を表す
// JSON: {"response": {...}} に対応
type APIResponse struct {
	PNResponse `json:"response"`
}

// PNResponse : log/YYYYMM.jsonlに記録するJSONの子要素
// slog.Any() に書き込める型
type PNResponse struct {
	Message string        `json:"msg"`              // ログの概要
	Error   []ErrorRecord `json:"errors,omitempty"` // エラーがあればErrorRecordを追記
	SHA256  string        `json:"sha256,omitempty"` // Sheet構造体から計算したsha256ハッシュ
	Sheet   input.Sheet   `json:"sheet,omitempty"`  // Sheet構造体のJSON
}

// ErrorRecord : エラーの詳細を保持するための構造体
type ErrorRecord struct {
	Message string `json:"message"`
	Err     error  `json:"-"` // 内部エラーの保持
	Details string `json:"details,omitempty"`
	Key     string `json:"key,omitempty"`
	Index   *int   `json:"index,omitempty"` // オプショナルなのでポインタ型
}

// // Error errorインターフェースを満たすための実装
// func (e *ErrorRecord) Error() string {
// 	return e.Message
// }
//
// // Unwrap : エラーチェーンをサポートするための実装
// func (e *ErrorRecord) Unwrap() error {
// 	return e.Err
// }

// NewErrorRecord : common.PNError
func NewErrorRecord(err error) *ErrorRecord {
	return &ErrorRecord{
		Message: err.Error(),
		Err:     err,
	}
}

// func (e *ErrorRecord) WithDetails(s string) *ErrorRecord {
// 	e.Details = s
// 	return e
// }
//
// func (e *ErrorRecord) WithKey(s string) *ErrorRecord {
// 	e.Key = s
// 	return e
// }
//
// func (e *ErrorRecord) WithIndex(i int) *ErrorRecord {
// 	e.Index = &i
// 	return e
// }

// handleResponse processes API responses based on status code
// codeに対する処理を分岐
// 200台ステータスコードは何もしない
// 300,400番台ステータスコードはファイル名.jsonにエラーの内容を書き込む
// 500番台ステータスコードはPOSTに失敗しているので、faital_report.log にエラーを書き込み
func JsonParse(body []byte) (*APIResponse, error) {
	// APIレスポンス解析とエラー出力
	if body == nil || len(body) < 1 {
		return nil, errors.New("bodyがありません")
	}
	// レスポンス解析
	var resp APIResponse
	err := json.Unmarshal(body, &resp)
	fmt.Printf("%s\n", body)
	if err != nil {
		return nil, fmt.Errorf("JSONパースに失敗しました: %s, %w", body, err)
	}
	// fmt.Printf("[DEBUG] pnresponse %#v\n", resp)
	return &resp, nil
}
