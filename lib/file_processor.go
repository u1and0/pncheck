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
		locationParts = append(locationParts, fmt.Sprintf("%d行目", *e.Index))
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

// collectValidationErrors はローカルとAPIの一次検証エラーを収集します
func collectValidationErrors(sheet *input.Sheet, resp *api.APIResponse, code int) (errs []string) {
	// ローカルでの検証
	if err := sheet.CheckSheetVersion(); err != nil {
		errs = append(errs, fmt.Sprintf("要求票の版番号の確認: %s", err))
	}
	if err := sheet.CheckOrderItemsSortOrder(); err != nil {
		errs = append(errs, fmt.Sprintf("入力Iが納期と品番順にソートされていません: %v", err))
	}

	// APIからのエラー
	if resp.Message != "" && code >= 400 {
		errs = append(errs, resp.Message)
	}
	for _, e := range resp.PNResponse.Error {
		errs = append(errs, formatErrorMessage(e))
	}
	return
}

// handleOverridePost はエラー時のオーバーライドPOST処理を実行します
func handleOverridePost(report *output.Report, sheet *input.Sheet) (errs []string) {
	sheet.Config.Overridable = true
	sheet.Config.Validatable = false
	body, code, err := sheet.Post()
	if err != nil {
		errs = append(errs, fmt.Sprintf("API通信エラー(2回目): %v", err))
		return
	}

	resp, err := api.JsonParse(body)
	if err != nil {
		errs = append(errs, fmt.Sprintf("APIレスポンス解析エラー(2回目): %v", err))
		return
	}

	// リンクをオーバーライド後のものに更新
	report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)

	// 2回目のレスポンスがワーニング(300番台)の場合、ステータスを更新しメッセージも追加
	if code >= 300 && code < 400 {
		report.StatusCode = output.StatusCode(code)
		if resp.Message != "" {
			errs = append(errs, resp.Message)
		}
		for _, e := range resp.PNResponse.Error {
			errs = append(errs, formatErrorMessage(e))
		}
	}
	return
}

func processFile(filePath string, resultChan chan<- output.Report) {
	var report output.Report
	report.Filename = filepath.Base(filePath)

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

	// 2. 1回目のPOST (バリデーション有効)
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

	// 3. エラー収集
	errs := collectValidationErrors(&sheet, resp, code)
	if code >= 400 && code < 500 && len(errs) > 0 {
		report.StatusCode = output.StatusCode(code)
		report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)
		report.ErrorMessages = errs
		resultChan <- report

	} else if code > 300 {
		report.StatusCode = output.StatusCode(code)
		// 4. 必要に応じて2回目のPOST (オーバーライド)
		errs := handleOverridePost(&report, &sheet)
		report.ErrorMessages = errs
	}

	// 5. 最終的なステータスコードの決定
	if len(report.ErrorMessages) == 0 {
		report.StatusCode = output.StatusCode(code)
	}

	resultChan <- report
}
