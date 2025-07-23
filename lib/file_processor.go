/*
渡されたExcelファイルパスを並列にPNSearch APIに渡して確認します。

並列処理を使ってExcelをSheet型に変換し、
PNSearch API から受け取ったJSONデータをReport型として格納します。
*/
package lib

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"pncheck/lib/api"
	"pncheck/lib/input"
	"pncheck/lib/output"
)

// ProcessExcelFile は、複数のExcelファイルを並列に処理し、その結果を返します。
func ProcessExcelFile(filePaths []string) (reports output.Reports) {
	fileChan := make(chan string, len(filePaths))
	for _, filePath := range filePaths {
		fileChan <- filePath
	}
	close(fileChan)

	numWorkers := runtime.NumCPU()
	sem := make(chan bool, numWorkers)

	resultChan := make(chan output.Report, len(filePaths))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for filePath := range fileChan {
				sem <- true
				processFile(filePath, resultChan)
				<-sem
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		reports.Classify(result)
	}

	return
}

// formatErrorMessage はErrorRecordを整形して文字列として返します。
func formatErrorMessage(e api.ErrorRecord) string {
	var parts []string
	if e.Details != "" {
		parts = append(parts, e.Details)
	}

	var locationParts []string
	if e.Index != nil {
		locationParts = append(locationParts, fmt.Sprintf("%d行目", *e.Index+1))
	}
	if e.Key != "" {
		locationParts = append(locationParts, e.Key)
	}

	if len(locationParts) > 0 {
		parts = append(parts, fmt.Sprintf("[%s]", strings.Join(locationParts, ":")))
	}

	if len(parts) > 0 {
		return fmt.Sprintf("%s: %s", e.Message, strings.Join(parts, " "))
	}
	return e.Message
}

func processFile(filePath string, resultChan chan<- output.Report) {
	var report output.Report
	report.Filename = filepath.Base(filePath)

	// ローカルでのファイル処理エラー。StatusCodeは0のまま => FatalItemに分類される
	sheet, err := input.ReadExcelToSheet(filePath)
	if err != nil {
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("Excel読み込みエラー: %v", err))
		resultChan <- report
		return
	}

	if err := input.ActivateOrderSheet(filePath); err != nil {
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("入力Iのアクティベーションエラー: %v", err))
		resultChan <- report
		return
	}

	// 1回目のPOST
	// オーバーライドを無効、バリデーションを有効にして
	// エラーを観測する
	sheet.Config.Validatable = true
	sheet.Config.Overridable = false
	body, code, err := sheet.Post()
	if err != nil {
		report.StatusCode = 500 // API通信自体が失敗した場合はFatal
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("API通信エラー: %v", err))
		resultChan <- report
		return
	}

	resp, err := api.JsonParse(body)
	if err != nil {
		report.StatusCode = 500 // レスポンス解析エラーもFatal
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("APIレスポンス解析エラー: %v", err))
		resultChan <- report
		return
	}

	// APIによる確認後の処理 (ここからはStatusCode 400を設定する)
	if err := input.CheckSheetVersion(filePath); err != nil {
		report.StatusCode = 400
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("要求票の版番号の確認: %s", err))
	}

	if err := input.CheckOrderItemsSortOrder(sheet); err != nil {
		report.StatusCode = 400
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("入力Iが納期と品番順にソートされていません: %v", err))
	}

	// APIからの全体エラーメッセージを追加
	if resp.Message != "" && code >= 400 {
		report.StatusCode = 400
		report.ErrorMessages = append(report.ErrorMessages, resp.Message)
	}
	// APIからの詳細なエラーメッセージを追加
	for _, e := range resp.PNResponse.Error {
		report.StatusCode = 400 // 詳細エラーがある場合も400番台とする
		report.ErrorMessages = append(report.ErrorMessages, formatErrorMessage(e))
	}

	report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)

	// APIエラー(400番台)があり、エラーメッセージが記録されている場合、オーバーライドを試みる
	if code >= 400 && code < 500 && len(report.ErrorMessages) > 0 {
		// 2回目のPOSTを実行し、オーバーライドされた結果のリンクと追加の警告メッセージを取得する
		sheet.Config.Overridable = true
		sheet.Config.Validatable = false
		body, code2, err := sheet.Post()
		if err != nil {
			report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("API通信エラー(2回目): %v", err))
		} else {
			resp2, err := api.JsonParse(body)
			if err != nil {
				report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("APIレスポンス解析エラー(2回目): %v", err))
			} else {
				// 2回目のPOSTでWarningになった場合、ステータスを更新する
				if code2 >= 300 && code2 < 400 {
					report.StatusCode = output.StatusCode(code2)
				}

				// リンクをオーバーライド後のものに更新
				report.Link = input.BuildRequestURL(resp2.PNResponse.SHA256)

				// 2回目のレスポンスがワーニング(300番台)の場合、そのメッセージも追加する
				if code2 >= 300 && code2 < 400 {
					if resp2.Message != "" {
						report.ErrorMessages = append(report.ErrorMessages, resp2.Message)
					}
					for _, e := range resp2.PNResponse.Error {
						report.ErrorMessages = append(report.ErrorMessages, formatErrorMessage(e))
					}
				}
			}
		}
	}

	// 全てのエラーチェックが終わった後、エラーメッセージがなければAPIのコードを最終ステータスとする
	if len(report.ErrorMessages) == 0 {
		report.StatusCode = output.StatusCode(code)
	}

	resultChan <- report
}
