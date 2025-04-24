package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// processExcelFile は1つのExcelファイルを処理し、その結果を FileProcessResult として返します。
// 内部でファイルの読み込み、JSON変換、API呼び出し、レスポンス処理を行います。
func processExcelFile(filePath string) (*ErrorOutput, error) {
	if pnsearchServerAddress == "" {
		return nil, errors.New("APIサーバーアドレスが設定されていません")
	}

	// Excel読み込み
	s, err := readExcelToSheet(filePath)
	if err != nil {
		return nil, fmt.Errorf("Excel読み込みエラー: %w", err)
	}

	// JSON変換
	jsonData, err := convertToJSON(s)
	if err != nil {
		return nil, fmt.Errorf("JSON変換エラー: %w", err)
	}

	// API呼び出し
	body, code, err := postToConfirmAPI(jsonData, pnsearchServerAddress)
	if err != nil {
		return nil, fmt.Errorf("API通信エラー: %w", err)
	}

	// 4. APIレスポンス解析とエラー出力 (ボディがあれば実行)
	if body == nil || len(body) < 1 {
		return nil, fmt.Errorf("APIレスポンス解析エラー (ステータス: %d): %w", code, err)
	}

	apiResponse, err := handleAPIResponse(body)
	if err != nil {
		return nil, err
	}
	log.Printf("pnsearch response: %#v\n", apiResponse)

	// JSONデコード成功
	// レスポンスのSheetとSHA256は捨てる
	baseName := filepath.Base(filePath)
	outputData := ErrorOutput{Filename: baseName, Msg: apiResponse.Message, Errors: apiResponse.Error}
	return &outputData, nil
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
