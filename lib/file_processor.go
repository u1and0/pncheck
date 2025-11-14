/*
渡されたExcelファイルパスを並列にPNSearch APIに渡して確認します。

並列処理を使ってExcelをSheet型に変換し、
PNSearch API から受け取ったJSONデータをReport型として格納します。
*/
package lib

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"pncheck/lib/api"
	"pncheck/lib/input"
	"pncheck/lib/output"
)

const (
	ProjectIDLength  = 12
	ProjectAssyDigit = 9
	ProjectAssyValue = 6
)

// ProcessExcelFile は、複数のExcelファイルを並列に処理し、その結果を返します。
//
// @errors:
//
//	Reports.Classify(): unknown status code %d: must 200 <= code < 600
func ProcessExcelFile(filePaths []string, debugLevel int) (output.Reports, error) {
	var (
		reports  output.Reports
		fileChan = make(chan string, len(filePaths))
	)

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
				processFile(filePath, resultChan, debugLevel)
				<-sem
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if err := reports.Classify(result); err != nil {
			return reports, err
		}
	}

	return reports, nil
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

// collectLocalErrors はローカルとAPIの一次検証エラーを収集します
func collectLocalErrors(sheet *input.Sheet) (errs []string) {
	// ローカルでの検証
	if err := sheet.CheckSheetVersion(); err != nil {
		errs = append(errs, fmt.Sprintf("要求票の版番号の確認: %s", err))
	}

	// 10桁目が6 == 組部品なのでソートチェックをしない
	if len(sheet.Header.ProjectID) < ProjectIDLength {
		errs = append(errs, fmt.Sprintf("製番の桁数が異常です。%s", sheet.Header.ProjectID))
	} else {
		class, err := strconv.Atoi(sheet.Header.ProjectID[ProjectAssyDigit : ProjectAssyDigit+1])
		if err != nil {
			errs = append(errs, fmt.Sprintf("製番の値が異常です。%s", sheet.Header.ProjectID))
		}
		if class != ProjectAssyValue {
			if err := sheet.CheckOrderItemsSortOrder(); err != nil {
				errs = append(errs, fmt.Sprintf("入力Iが納期と品番順にソートされていません: %v", err))
			}
		}
	}
	return
}

// handleOverridePost はエラー時のオーバーライドPOST処理を実行し、
// reportを完全に更新します
func handleOverridePost(report *output.Report, sheet *input.Sheet) error {
	sheet.Config.Overridable = true  // サーバー側の自動更新を許可
	sheet.Config.Validatable = false // あえてワーニングを表示するためエラーチェック無効化
	body, code, err := sheet.Post()
	if err != nil {
		return fmt.Errorf("API通信エラー(2回目): %v", err)
	}

	resp, err := api.JSONParse(body)
	if err != nil {
		return fmt.Errorf("APIレスポンス解析エラー(2回目): %v", err)
	}

	// reportを完全に更新
	report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)
	report.StatusCode = output.StatusCode(code)

	// エラーメッセージを直接設定
	var errs []string
	if resp.Message != "" {
		errs = append(errs, resp.Message)
	}
	for _, e := range resp.PNResponse.Error {
		errs = append(errs, formatErrorMessage(e))
	}
	report.ErrorMessages = errs

	return nil
}

func processFile(filePath string, resultChan chan<- output.Report, debugLevel int) {
	var report output.Report
	report.Filename = filepath.Base(filePath)

	sheet, err := input.ReadExcelToSheet(filePath)
	if err != nil {
		report.StatusCode = 500
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("Excel読み込みエラー: %v", err))
		resultChan <- report
		return
	}

	// Debug Print: Excel parse, API request
	if debugLevel > 2 {
		jsonData, err := json.MarshalIndent(sheet, "", "  ")
		if err != nil {
			err = fmt.Errorf("Sheet構造体のJSON変換に失敗しました: %w", err)
			return
		}
		fmt.Printf("%s\n", jsonData)
	}

	if err := input.ActivateOrderSheet(filePath); err != nil {
		report.StatusCode = 500
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("入力Iのアクティベーションエラー: %v", err))
		resultChan <- report
		return
	}

	// 2. 1回目のPOST
	sheet.Config.Validatable = true  // エラーチェック有効化
	sheet.Config.Overridable = false // サーバー側の自動更新を無効化
	body, code, err := sheet.Post()
	if err != nil {
		report.StatusCode = 500 // API通信自体が失敗した場合はFatal
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("API通信エラー: %v", err))
		resultChan <- report
		return
	}

	// Debug Print API response
	if debugLevel > 1 {
		fmt.Printf("%s\n", body)
	}

	resp, err := api.JSONParse(body)
	if err != nil {
		report.StatusCode = 500 // レスポンス解析エラーもFatal
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("APIレスポンス解析エラー: %v", err))
		resultChan <- report
		return
	}

	// 3. エラー収集
	errs := collectLocalErrors(&sheet)
	if errs != nil {
		report.StatusCode = 500
	} else {
		report.StatusCode = output.StatusCode(code)
	}

	// 4. APIからのエラー
	if resp.Message != "" && code >= 400 {
		errs = append(errs, resp.Message)
	}
	for _, e := range resp.PNResponse.Error {
		errs = append(errs, formatErrorMessage(e))
	}
	report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)
	report.ErrorMessages = errs

	// 1回目のレポート送信
	// 300番台以下：Warning/Success - そのまま返す
	resultChan <- report

	// 400番台: Error処理で1回目のレポートを送信後、2回目のPOSTを実行
	if code >= 400 && code < 500 {
		secondReport := output.Report{
			Filename: filepath.Base(filePath),
		}
		// 2回目のPOST (オーバーライド)
		if err := handleOverridePost(&secondReport, &sheet); err != nil {
			// システムエラーの場合
			secondReport.StatusCode = 500
			secondReport.ErrorMessages = []string{err.Error()}
		}
		resultChan <- secondReport
	}
}
