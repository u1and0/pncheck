package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"pncheck/lib"
)

const (
	VERSION    = "v1.5.0"
	outputPath = "pncheck_report.html" // エラー出力ファイル
)

var (
	BuildTime string
)

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatalln(err)
	}

	// 各ファイルを処理
	reports := lib.ProcessExcelFile(filePaths)
	reports.Version = VERSION
	reports.BuildTime = BuildTime
	reports.ExecutionTime = time.Now().Format("2006/01/02 15:04:05")

	fmt.Println(reports)
	if err = reports.Publish(outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "レポートファイルの出力に失敗しました: %v\n", err)
	}
}
