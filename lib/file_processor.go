package lib

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"pncheck/lib/api"
	"pncheck/lib/input"
	"pncheck/lib/output"
)

// ProcessExcelFile は、複数のExcelファイルを並列に処理し、その結果を返します。
func ProcessExcelFile(filePaths []string) output.Reports {
	var reports output.Reports
	reports.ExecutionTime = time.Now().Format("2006/01/02 15:04:05")

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

	return reports
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

	if err := input.CheckOrderItemsSortOrder(sheet); err != nil {
		report.ErrorMessage = fmt.Sprintf("入力Iが納期と品番順にソートされていません: %v", err)
		resultChan <- report
		return
	}

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

	report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)
	report.StatusCode = output.StatusCode(code)

	if code >= 500 {
		report.ErrorMessage = resp.Message
	}
	resultChan <- report
}
