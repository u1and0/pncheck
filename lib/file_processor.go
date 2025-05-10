package lib

import (
	"fmt"
	"os"

	"pncheck/lib/input"
	"pncheck/lib/output"
)

// これより大きいHTTPステータスコードは処理を分岐する
// 逆に、successCode未満のステータスは成功
const (
	successCode = 300
	errorCode   = 500
)

// ProcessExcelFile は1つのExcelファイルを処理し、その結果を FileProcessResult として返します。
// 内部でファイルの読み込み、JSON変換、API呼び出し、レスポンス処理を行います。
func ProcessExcelFile(filePath string) error {
	// Excel読み込み
	sheet, err := input.ReadExcelToSheet(filePath)
	if err != nil {
		return fmt.Errorf("Excel読み込みエラー: %w", err)
	}

	// API呼び出し
	body, code, err := sheet.Post()
	if err != nil {
		return fmt.Errorf("API通信エラー: %w", err)
	}
	return handleResponse(filePath, body, code)
}

// handleResponse processes API responses based on status code
// codeに対する処理を分岐
// 200台ステータスコードは何もしない
// 300,400番台ステータスコードはファイル名.jsonにエラーの内容を書き込む
// 500番台ステータスコードはPOSTに失敗しているので、faital_report.log にエラーを書き込み
func handleResponse(filePath string, body []byte, code int) error {
	// APIレスポンス解析とエラー出力 (ボディがあれば実行)
	if body == nil || len(body) < 1 {
		return fmt.Errorf("APIレスポンス解析エラー bodyがありません(ステータス: %d)", code)
	}
	// 500番台はfatal_report_log.json にエラーを追記する
	if code >= errorCode {
		return fmt.Errorf("APIレスポンス解析エラー (ステータス: %d): %s", code, body)
	}

	// 300,400番台はPNResponseをファイル名+.jsonに書き込む
	if code >= successCode {
		// 標準エラーにResponse とステータスコード、
		// 標準出力にレスポンス詳細を出力することで
		// `pncheck XYZ.xlsx | jq`
		// のようにしてJSONの整形ができる
		fmt.Fprintf(os.Stderr, "PNSerach response %d\n", code)
		fmt.Printf("%s\n", body)

		jsonFilename := input.FilenameWithoutExt(filePath) + ".json"
		return output.WriteErrorToJSON(jsonFilename, body)
	}
	// 成功したらコンソールに成功メッセージを書くだけ
	fmt.Fprintln(os.Stderr, "Success:", filePath)
	return nil
}
