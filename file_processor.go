package main

import (
	"encoding/json"
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
func processExcelFile(filePath string) ([]byte, error) {
	// 1. Excel読み込み
	sheetData, err := readExcelToSheet(filePath)
	if err != nil {
		return nil, fmt.Errorf("Excel読み込みエラー: %w", err)
	}

	// 2. JSON変換
	jsonData, err := convertToJSON(sheetData)
	if err != nil {
		return nil, fmt.Errorf("JSON変換エラー: %w", err)
	}

	// 3. API呼び出し
	if pnsearchServerAddress == "" {
		return nil, errors.New("APIサーバーアドレスが設定されていません")
	}

	responseBody, statusCode, err := postToConfirmAPI(jsonData, pnsearchServerAddress)
	if err != nil {
		return nil, fmt.Errorf("API通信エラー: %w", err)
	}

	// 4. APIレスポンス解析とエラー出力 (ボディがあれば実行)
	if responseBody == nil || len(responseBody) < 1 {
		return nil, fmt.Errorf("APIレスポンス解析エラー (ステータス: %d): %w", statusCode)
	}

	apiResponse, err := handleAPIResponse(responseBody)
	if err != nil {
		return nil, err
	}

	// JSONデコード成功
	// レスポンスのSheetとSHA256は捨てる
	baseName := filepath.Base(filePath)
	outputData := ErrorOutput{Filename: baseName, Msg: apiResponse.Message, Errors: apiResponse.Error}
	jsonBytes, err := json.MarshalIndent(outputData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf(
			"ERROR processing %s: エラー情報のJSONマーシャリングに失敗: %v",
			filePath,
			err,
		)
	}
	return jsonBytes, nil
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
