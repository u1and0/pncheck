package main

import (
	"fmt"
	"log"
	"os"

	"pncheck/lib"
)

const (
	errorReportFileName = "error_report_log.json" // エラーレポートのデフォルトファイル名
	successDirName      = "success"               // 成功ファイルを移動するディレクトリ名
)

func main() {
	// 1. コマンドライン引数を解析
	filePaths, err := lib.ParseArguments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1) // 引数エラーは終了コード1
	}

	// success ディレクトリ作成
	err = os.MkdirAll(successDirName, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: success ディレクトリ作成失敗: %v\n", err)
	}

	// 各ファイルを処理
	for _, filePath := range filePaths {
		eo, err := lib.ProcessExcelFile(filePath)
		log.Printf("process output: %#v\n", eo)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			// エラーがあればerror_reportに記録して次の処理へ
			err = eo.WriteErrorFile(errorReportFileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			continue
		}
		// エラーがなければsuccessディレクトリに移動して次のファイル処理へ
		err = lib.MoveFileToSuccess(filePath, successDirName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
}
