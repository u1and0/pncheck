package main

import (
	"errors"
	"fmt"
	"log" // logErrorで使用想定
	"os"
	"path/filepath"
	// logErrorで使用想定
	// logErrorで使用想定
)

// --- グローバル変数 (ビルド時注入想定) ---
var pnsearchServerAddress string // 例: "http://localhost:8080" (ビルド時に注入)

// processExcelFile は1つのExcelファイルを処理し、その結果を FileProcessResult として返します。
// 内部でファイルの読み込み、JSON変換、API呼び出し、レスポンス処理を行います。
func processExcelFile(filePath string) FileProcessResult {
	result := FileProcessResult{FilePath: filePath}

	// 1. Excelファイルを読み込む
	sheetData, err := readExcelToSheet(filePath)
	if err != nil {
		result.IsSuccess = false
		result.ProcessError = fmt.Errorf("Excel読み込みエラー: %w", err)
		logError(result.ProcessError, filePath) // エラーログ出力
		return result                           // エラー発生時はここで処理終了
	}
	log.Printf("[DEBUG]edSheet data: %#v\n", sheetData)

	// 2. 読み込んだデータをAPI仕様のJSONに変換する
	jsonData, err := convertToJSON(sheetData)
	if err != nil {
		result.IsSuccess = false
		result.ProcessError = fmt.Errorf("JSON変換エラー: %w", err)
		logError(result.ProcessError, filePath)
		return result
	}
	log.Printf("[DEBUG] Parsed JSON: %#v\n", string(jsonData))

	// 3. APIにJSONデータをPOSTする (サーバーアドレスが必要)
	if pnsearchServerAddress == "" {
		result.IsSuccess = false
		err = errors.New("APIサーバーアドレスが設定されていません (ビルド時に -ldflags で指定)")
		result.ProcessError = err
		logError(result.ProcessError, filePath)
		return result
	}
	responseBody, err := postToConfirmAPI(jsonData, pnsearchServerAddress)
	if err != nil {
		result.IsSuccess = false
		result.ProcessError = fmt.Errorf("API通信エラー: %w", err)
		logError(result.ProcessError, filePath)
		return result
	}

	// 4. APIレスポンスを解析・処理する
	apiResponse, err := handleAPIResponse(responseBody)
	if err != nil {
		result.IsSuccess = false
		result.ProcessError = fmt.Errorf("APIレスポンス解析エラー: %w", err)
		logError(result.ProcessError, filePath)
		return result
	}

	// 5. APIレスポンスから得られた結果を結果構造体に格納
	result.IsSuccess = true                             // ここまで来たらプロセス自体は成功
	result.ValidationError = len(apiResponse.Error) > 0 // APIエラーが1つ以上あれば検証エラー
	result.ApiErrors = apiResponse.Error

	// 6. 最終的な結果構造体を返す
	return result
}

// logError関数: 実行中のエラーを標準エラー出力に記録する（ヘルパー関数）
// (以前の構成案から実装)
func logError(err error, context string) {
	// 現在時刻などを付与して、標準エラー出力 (os.Stderr) に整形して出力
	// log パッケージを使うと便利
	log.Printf("ERROR processing %s: %v\n", context, err)
	// fmt.Fprintf(os.Stderr, "[%s] ERROR processing %s: %v\n", time.Now().Format(time.RFC3339), context, err)
}

// moveFileToSuccess は指定されたファイルを指定されたディレクトリに移動します。
func moveFileToSuccess(filePath string, destDir string) error {
	// 移動先ディレクトリが存在するか確認
	destInfo, err := os.Stat(destDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("移動先ディレクトリ '%s' が存在しません", destDir)
	} else if err != nil {
		return fmt.Errorf("移動先ディレクトリ '%s' の状態確認中にエラー: %w", destDir, err)
	}
	// 移動先がディレクトリでない場合もエラー
	if !destInfo.IsDir() {
		return fmt.Errorf("移動先パス '%s' はディレクトリではありません", destDir)
	}

	baseName := filepath.Base(filePath)
	destPath := filepath.Join(destDir, baseName)

	// 移動先に同名ファイルが存在するかチェック (上書き防止)
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("移動先に同名ファイル '%s' が既に存在します (%s)", baseName, destPath)
	} else if !os.IsNotExist(err) {
		// Statで予期せぬエラー (権限など)
		return fmt.Errorf("移動先のファイル存在チェック中にエラー (%s): %w", destPath, err)
	}

	// ファイルを移動 (os.Rename)
	err = os.Rename(filePath, destPath)
	if err != nil {
		// Renameのエラーには様々な原因が含まれる (ファイル無し、権限、別デバイスなど)
		return fmt.Errorf("os.Rename 実行中にエラー (%s -> %s): %w", filePath, destPath, err)
	}
	return nil // 正常終了
}
