package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"pncheck/lib/input"
	"pncheck/lib/output"
)

// サーバーからのエラー出力のうち、このコマンドで使用するもの
type ErrorOutput struct {
	Filename string               `json:"filename"`
	Msg      string               `json:"msg"`
	Errors   []output.ErrorRecord `json:"errors,omitempty"`
}

// WriteErrorFile は集約されたエラー結果を指定されたファイルにJSON形式で出力します。
// エラーがない場合はファイルを作成しません。
// 出力形式: エラーがあった FileProcessResult のスライスをインデント付きJSONで出力
func (eo *ErrorOutput) WriteErrorFile(path string) error {
	// ファイルを開く (なければ作成、あれば上書き)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' の作成に失敗しました: %w", path, err)
	}
	defer file.Close()

	b, err := json.MarshalIndent(&eo, "", "  ")
	_, err = file.Write(b)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' へのJSONデータ書き込みに失敗しました: %w", path, err)
	}
	return nil
}

// ProcessExcelFile は1つのExcelファイルを処理し、その結果を FileProcessResult として返します。
// 内部でファイルの読み込み、JSON変換、API呼び出し、レスポンス処理を行います。
func ProcessExcelFile(filePath string) (*ErrorOutput, error) {
	// Excel読み込み
	sheet, err := input.ReadExcelToSheet(filePath)
	if err != nil {
		return nil, fmt.Errorf("Excel読み込みエラー: %w", err)
	}

	// API呼び出し
	body, code, err := sheet.Post()
	if err != nil {
		return nil, fmt.Errorf("API通信エラー: %w", err)
	}

	// 4. APIレスポンス解析とエラー出力 (ボディがあれば実行)
	if err!=nil || body == nil || len(body) < 1 {
		return nil, fmt.Errorf("APIレスポンス解析エラー (ステータス: %d): %w", code, err)
	}

	apiResponse, err := output.HandleAPIResponse(body)
	if err != nil {
		return nil, err
	}
	log.Printf("pnsearch response: %#v\n", apiResponse)

	// JSONデコード成功
	// レスポンスのSheetとSHA256は捨てる
	baseName := filepath.Base(filePath)
	outputData := ErrorOutput{
		Filename: baseName,
		Msg:      apiResponse.Message,
		Errors:   apiResponse.Error,
	}
	return &outputData, nil
}

// MoveFileToSuccess は指定されたファイルを指定されたディレクトリに移動します。
func MoveFileToSuccess(filePath string, destDir string) error {
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
