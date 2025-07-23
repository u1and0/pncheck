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

func processFile(filePath string, resultChan chan<- output.Report) {
	var report output.Report
	report.Filename = filepath.Base(filePath)

	sheet, err := input.ReadExcelToSheet(filePath)
	if err != nil {
		report.ErrorMessage = fmt.Sprintf("Excel読み込みエラー: %v", err)
		resultChan <- report
		return
	}

	if err := input.ActivateOrderSheet(filePath); err != nil {
		report.ErrorMessage = fmt.Sprintf("入力Iのアクティベーションエラー: %v", err)
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
		report.ErrorMessage = fmt.Sprintf("API通信エラー: %v", err)
		resultChan <- report
		return
	}

	resp, err := api.JsonParse(body)
	if err != nil {
		report.ErrorMessage = fmt.Sprintf("APIレスポンス解析エラー: %v", err)
		resultChan <- report
		return
	}

	// APIによる確認後の処理
	if err := input.CheckSheetVersion(filePath); err != nil {
		report.StatusCode = 400
		report.ErrorMessage = fmt.Sprintf("要求票の版番号の確認: %s", err)
	} else {
		report.StatusCode = output.StatusCode(code)
	}

	if err := input.CheckOrderItemsSortOrder(sheet); err != nil {
		report.StatusCode = 400
		report.ErrorMessage = fmt.Sprintf("入力Iが納期と品番順にソートされていません: %v", err)
	} else {
		report.StatusCode = output.StatusCode(code)
	}

	report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)
	if code >= 500 {
		report.ErrorMessage = resp.Message
	} else if code >= 400 {
		// 1回目POSTの結果を保存
		resultChan <- report

		// 2回目のPOSTを実行
		// httpステータス400以上でエラーが含まれる場合は
		// ワーニングを表示したいので
		// オーバーライドを有効、 バリデーションを無効にして
		sheet.Config.Overridable = true
		sheet.Config.Validatable = false
		body, code, _ := sheet.Post()
		resp, err := api.JsonParse(body)
		if err != nil {
			report.ErrorMessage = fmt.Sprintf("APIレスポンス解析エラー: %v", err)
			resultChan <- report
			return
		}

		// レスポンスがワーニングでなければ何もせずに終了
		if code < 300 {
			return
		}
		report.StatusCode = output.StatusCode(code)
		report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)
	}
	resultChan <- report
}
