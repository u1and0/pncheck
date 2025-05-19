package main

import (
	"fmt"
	"log"
	"os"

	"pncheck/lib"
	"pncheck/lib/output"
)

const (
	VERSION    = "v0.1.1r"
	FATALLOG   = "pncheck_fatal_report.log"
	outputPath = "pncheck_report.html"
)

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatalln(err)
		output.LogFatalError(FATALLOG, err.Error())
	}

	// 各ファイルを処理
	reports := lib.ProcessExcelFile(filePaths)
	if err = reports.Publish(outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "レポートファイルの出力に失敗しました: %v\n", err)
	}
}
