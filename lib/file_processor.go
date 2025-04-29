package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pncheck/lib/input"
)

const (
	fatalLog = "fatal_report_log.json" // エラーレポートのデフォルトファイル名
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
	// APIレスポンス解析とエラー出力 (ボディがあれば実行)
	if body == nil || len(body) < 1 {
		return fmt.Errorf("APIレスポンス解析エラー (ステータス: %d): %w", code, err)
	}

	fmt.Printf("Response: %d: %s", code, body)

	// codeに対する処理を分岐
	// 200で成功なら何もしない
	// 300-400番台ステータスコードはファイル名.jsonにエラーの内容を書き込む
	// 500番台ステータスコードはPOSTに失敗しているので、faital_report.log にエラーを書き込み
	switch {
	// 成功したらコンソールに成功メッセージを書くだけ
	case code < 300:
		fmt.Println("Success:", filePath)

	// 300,400番台はPNResponseをJSONに書き込む
	case code < 500:
		jsonPath := filepath.Base(filePath) + ".json"
		f, err := os.Create(jsonPath)
		if err != nil {
			return fmt.Errorf("エラーファイル '%s' の作成に失敗しました: %w", jsonPath, err)
		}
		defer f.Close()

		_, err = f.Write(body)
		if err != nil {
			return fmt.Errorf("エラーファイル '%s' へのJSONデータ書き込みに失敗しました: %w", jsonPath, err)
		}

	// fatal_report.logを開いて、エラーの内容を追記する。
	// すでにfatal_report.logが存在しても初期化しないで内容を追記する
	// ファイルが存在する場合、内容を読み取る
	default:
		var data []byte
		if _, err := os.Stat(fatalLog); err == nil {
			data, err = os.ReadFile(fatalLog)
			if err != nil {
				return err
			}
		}

		// エラーの内容を追記
		now := time.Now().Format("2006/01/02 15:04:05")
		msg := now + ": PNSearch /confirm API への通信に失敗しました。\n"
		if data != nil {
			data = append(data, []byte(msg)...)
		} else {
			data = []byte(msg)
		}

		// ファイルに書き込み
		return os.WriteFile(fatalLog, data, 0644)
	}

	return nil
}
