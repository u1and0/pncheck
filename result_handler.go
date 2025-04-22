package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// aggregateResults は複数ファイルの処理結果を集約し、AggregatedResult構造体で返します。
func aggregateResults(results []FileProcessResult) AggregatedResult {
	var aggregated AggregatedResult
	aggregated.TotalFiles = len(results)
	aggregated.ErrorDetails = make([]FileProcessResult, 0) // エラー詳細リストを初期化

	for _, r := range results {
		if r.IsSuccess {
			aggregated.SuccessFiles++
			if r.ValidationError {
				aggregated.InvalidFiles++
				// 検証エラーの詳細をリストに追加
				aggregated.ErrorDetails = append(aggregated.ErrorDetails, r)
			} else {
				aggregated.ValidFiles++
			}
		} else {
			aggregated.ProcessErrorFiles++
			// プロセスエラーの詳細をリストに追加
			aggregated.ErrorDetails = append(aggregated.ErrorDetails, r)
		}
	}
	return aggregated
}

// writeErrorFile は集約されたエラー結果を指定されたファイルに人間が読みやすい形式で出力します。
// エラーがない場合はファイルを作成しません。
// フォーマット: TSV (タブ区切り)
// ファイルパス\tエラー種別\tエラー内容\t[エラー箇所キー]\t[エラー箇所詳細]\t[エラー行]
func writeErrorFile(aggregatedResult AggregatedResult, outputFilePath string) error {
	// エラーがなければファイルを作成しない
	if len(aggregatedResult.ErrorDetails) == 0 {
		return nil
	}
	if outputFilePath == "" {
		return errors.New("エラーファイルの出力パスが指定されていません")
	}

	// ファイルを開く (なければ作成、あれば上書き)
	file, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' の作成に失敗しました: %w", outputFilePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush() // 関数終了時にバッファの内容を確実に書き込む

	// ヘッダー行を書き込む (オプション)
	_, err = writer.WriteString("ファイルパス\tエラー種別\tエラー内容\tエラー箇所キー\tエラー箇所詳細\tエラー行\n")
	if err != nil {
		return fmt.Errorf("エラーファイルヘッダーの書き込みに失敗しました: %w", err)
	}

	// エラー詳細をループして書き込む
	for _, detail := range aggregatedResult.ErrorDetails {
		if detail.ProcessError != nil {
			// プロセスエラーの場合
			line := fmt.Sprintf("%s\tプロセスエラー\t%s\t-\t-\t-\n",
				detail.FilePath,
				escapeTsv(detail.ProcessError.Error()), // エラーメッセージ中のタブや改行をエスケープ
			)
			if _, err := writer.WriteString(line); err != nil {
				return fmt.Errorf("プロセスエラー情報のファイル書き込みに失敗しました (%s): %w", detail.FilePath, err)
			}
		} else if detail.ValidationError && len(detail.ApiErrors) > 0 {
			// 検証エラーの場合 (APIエラーが1つ以上ある)
			for _, apiErr := range detail.ApiErrors {
				rowIndex := "-"
				if apiErr.Index != nil {
					rowIndex = strconv.Itoa(*apiErr.Index)
				}
				detailsStr := "-" // Detailsが空の場合に "-" を使う
				if apiErr.Details != "" {
					detailsStr = escapeTsv(apiErr.Details)
				}

				line := fmt.Sprintf("%s\t検証エラー\t%s\t%s\t%s\t%s\n",
					detail.FilePath,
					escapeTsv(apiErr.Message),
					escapeTsv(apiErr.Key),
					detailsStr, // 修正した detailsStr を使用
					rowIndex,
				)
				if _, err := writer.WriteString(line); err != nil {
					return fmt.Errorf("検証エラー情報のファイル書き込みに失敗しました (%s, Key: %s): %w", detail.FilePath, apiErr.Key, err)
				}
			}
		}
		// ProcessError も ValidationError もないが ErrorDetails に含まれるケースは想定しない
	}

	return nil // 正常終了
}

// escapeTsv は文字列中のタブ文字と改行文字をエスケープします。
// TSVファイルで正しく表示するため。
func escapeTsv(s string) string {
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}
