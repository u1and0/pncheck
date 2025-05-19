package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"pncheck/lib/input"
)

// これより大きいHTTPステータスコードは処理を分岐する
// 逆に、successCode未満のステータスは成功
const (
	successCode = 300
	errorCode   = 500
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

type ErrorRecord struct {
	Message string `json:"message"`
	Err     error  `json:"-"` // 内部エラーの保持
	Details string `json:"details,omitempty"`
	Key     string `json:"key,omitempty"`
	Index   *int   `json:"index,omitempty"` // オプショナルなのでポインタ型
}

// ProcessExcelFile は1つのExcelファイルを処理し、その結果を FileProcessResult として返します。
// 内部でファイルの読み込み、JSON変換、API呼び出し、レスポンス処理を行います。
func ProcessExcelFile(filePath string) error {
	// Excel読み込み
	sheet, err := input.ReadExcelToSheet(filePath)
	if err != nil {
		return fmt.Errorf("Excel読み込みエラー: %w", err)
	}

	// 入力Iをアクティベートする
	err = input.ActivateOrderSheet(filePath)
	if err != nil {
		return fmt.Errorf("Excel読み込みエラー: %w", err)
	}

	// API呼び出し
	body, code, err := sheet.Post()
	if err != nil {
		return fmt.Errorf("API通信エラー: %w", err)
	}
	if code >= errorCode {
		return fmt.Errorf("API通信エラー: ステータス: %d: %s", code, body)
	}
	fmt.Fprintf(os.Stderr, "code: %d\n", code)
	resp, err := jsonParse(body)
	if err != nil {
		return fmt.Errorf("APIレスポンス解析エラー: %w", err)
	}

	// 300,400番台はPNResponseをファイル名+.jsonに書き込む
	if code >= successCode {
		url := fmt.Sprintf("%s/index?hash=%s#requirement-tab", input.ServerAddress, resp.SHA256)
		return fmt.Errorf("%s, 詳細は\n%s\nを確認してください\n", resp.Message, url)
	}
	return nil
}

// handleResponse processes API responses based on status code
// codeに対する処理を分岐
// 200台ステータスコードは何もしない
// 300,400番台ステータスコードはファイル名.jsonにエラーの内容を書き込む
// 500番台ステータスコードはPOSTに失敗しているので、faital_report.log にエラーを書き込み
func jsonParse(body []byte) (*APIResponse, error) {
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
