package lib

import (
	"fmt"
	"os"

	"pncheck/lib/input"
	"pncheck/lib/output"
)

// これ未満のHTTPステータスコードはswitchに書く処理を行う
const (
	successCode = 400
	errorCode   = 500
)

// ProcessExcelFile は1つのExcelファイルを処理し、その結果を FileProcessResult として返します。
// 内部でファイルの読み込み、JSON変換、API呼び出し、レスポンス処理を行います。
func ProcessExcelFile(filePath string) error {
	// 渡されたファイルがディレクトリの場合は無視
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("ファイル情報読み込みエラー: %w", err)
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("%s はディレクトリです\n", filePath)
	}
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
// 200-300台ステータスコードは何もしない
// 400番台ステータスコードはファイル名.jsonにエラーの内容を書き込む
// 500番台ステータスコードはPOSTに失敗しているので、faital_report.log にエラーを書き込み
func handleResponse(filePath string, body []byte, code int) error {
	// APIレスポンス解析とエラー出力 (ボディがあれば実行)
	if body == nil || len(body) < 1 {
		return fmt.Errorf("APIレスポンス解析エラー bodyがありません(ステータス: %d)", code)
	}
	if code >= errorCode {
		return fmt.Errorf("APIレスポンス解析エラー (ステータス: %d): %s", code, body)
	}

	// 400番台はPNResponseをJSONに書き込む
	if code >= successCode {
		// TODO
		// 警告の場合はJSON?コンソールに成功メッセージを書くだけ？
		// case code < 400:
		// 	fmt.Println("Warning:", filePath)

		jsonFilename := input.FilenameWithoutExt(filePath) + ".json"
		return output.WriteErrorToJSON(jsonFilename, body)
	}
	// 成功したらコンソールに成功メッセージを書くだけ
	fmt.Println("Success:", filePath)
	return nil
}
