package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pncheck/lib/input"
	"pncheck/lib/output"
)

// これより大きいHTTPステータスコードは処理を分岐する
// 逆に、successCode未満のステータスは成功
const (
	successCode = 300
	errorCode   = 500
)

// ProcessExcelFile は1つのExcelファイルを処理し、その結果を FileProcessResult として返します。
// 内部でファイルの読み込み、JSON変換、API呼び出し、レスポンス処理を行います。
func ProcessExcelFile(filePaths []string) output.Reports {
	var reports output.Reports
	reports.ExecutionTime = time.Now().Format("2006/01/02 15:04:05")

	for _, filePath := range filePaths {
		var report output.Report
		report.Filename = filepath.Base(filePath)

		// Excel読み込み
		sheet, err := input.ReadExcelToSheet(filePath)
		if err != nil {
			report.ErrorMessage = fmt.Sprintf("Excel読み込みエラー", err)
			reports.FatalItems = append(reports.FatalItems, report)
			continue
		}

		// 入力Iをアクティベートして上書き保存する
		err = input.ActivateOrderSheet(filePath)
		if err != nil {
			report.ErrorMessage = fmt.Sprintf("Excel読み込みエラー", err)
			reports.FatalItems = append(reports.FatalItems, report)
			continue
		}

		// API呼び出し
		body, code, err := sheet.Post()
		if err != nil {
			report.ErrorMessage = fmt.Sprintf("API通信エラー", err)
			reports.FatalItems = append(reports.FatalItems, report)
			continue
		}
		fmt.Fprintf(os.Stderr, "code: %d\n", code)
		resp, err := output.JsonParse(body)
		if err != nil {
			report.ErrorMessage = fmt.Sprintf("APIレスポンス解析エラー: %v", err)
			reports.FatalItems = append(reports.FatalItems, report)
			continue
		}

		if code < 200 { // 発生しないはず
			report.ErrorMessage = fmt.Sprintf("不明なステータスコード: %v", err)
			reports.FatalItems = append(reports.FatalItems, report)
			continue
		}

		// 正常にAPIからのレスポンスを受け取った場合
		report.Link = input.BuildRequestURL(resp.PNResponse.SHA256)
		if code >= 500 {
			report.ErrorMessage = resp.Message
			reports.FatalItems = append(reports.FatalItems, report)
		} else if code >= 400 {
			reports.ErrorItems = append(reports.ErrorItems, report)
		} else if code >= 300 {
			reports.WarningItems = append(reports.WarningItems, report)
		} else if code >= 200 {
			reports.SuccessItems = append(reports.SuccessItems, report)
		}
	}
	return reports
}
