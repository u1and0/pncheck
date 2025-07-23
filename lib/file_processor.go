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

	// ローカルエラーなのでステータスコード0(初期値)
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

	// APIによる確認後の処理
	if err := input.CheckSheetVersion(filePath); err != nil {
		report.StatusCode = 400
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("要求票の版番号の確認: %s", err))
	}

	if err := input.CheckOrderItemsSortOrder(sheet); err != nil {
		report.StatusCode = 400
		report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("入力Iが納期と品番順にソートされていません: %v", err))
	}

	// APIからのエラーメッセージを追加
	if resp.Message != "" && code >= 400 {
		report.StatusCode = 400
		report.ErrorMessages = append(report.ErrorMessages, resp.Message)
	}

	// ローカルでの検証エラーがなく、APIからもエラーがなければ、APIのステータスコードを正式なものとして採用
	if len(report.ErrorMessages) == 0 {
		report.StatusCode = output.StatusCode(code)
	}

	report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)

	// APIエラー(400番台)があり、かつローカル検証エラーがなかった場合のみ、
	// オーバーライドを試みる
	if code >= 400 && code < 500 && len(report.ErrorMessages) > 0 {
		// 2回目のPOSTを実行し、オーバーライドされた結果のリンクを取得する
		sheet.Config.Overridable = true
		sheet.Config.Validatable = false
		body, _, err := sheet.Post() // 2回目のcodeはここでは不要
		if err != nil {
			report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("APIレスポンス解析エラー(2回目): %v", err))
		} else {
			resp, err := api.JsonParse(body)
			if err != nil {
				report.ErrorMessages = append(report.ErrorMessages, fmt.Sprintf("APIレスポンス解析エラー(2回目): %v", err))
			} else {
				// リンクをオーバーライド後のものに更新
				report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)
			}
		}
	}

	resultChan <- report
}
