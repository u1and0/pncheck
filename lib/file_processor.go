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

	// 200-400番台以外はサーバーエラーとして
	// fatal_report.logにエラーの内容を追記する。
	default:
		now := time.Now().Format("2006/01/02 15:04:05")
		msg := now + ": PNSearch /confirm API への通信に失敗しました。\n"
		return appendToFile(fatalLog, msg)
	}

	return nil
}

// appendToFile : ファイルの末尾に書き込む
// O_APPEND: ファイルの末尾に書き込む
// O_CREATE: ファイルが存在しない場合は作成する
// O_WRONLY: 書き込み専用で開く
func appendToFile(filePath string, msg string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(msg)
	return err
}
