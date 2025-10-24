package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"pncheck/lib"
	"pncheck/lib/input"
)

const (
	VERSION    = "v1.6.2"
	outputPath = "pncheck_report.html" // エラー出力ファイル
)

func main() {
	// コマンドライン引数を解析
	filePaths, verboseLevel, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatalln(err)
	}

	// 各ファイルを処理
	reports, err := lib.ProcessExcelFile(filePaths, verboseLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "レポートファイルの出力に失敗しました: %v\n", err)
	}
	reports.Version = VERSION
	reports.BuildTime = input.BuildTime
	reports.ExecutionTime = time.Now().Format("2006/01/02 15:04:05")

	if verboseLevel > 0 {
		b, err := reports.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSONの標準出力に失敗しました: %v\n", err)
		}
		fmt.Printf("%s\n", string(b)) // 標準出力
	}

	err = reports.Publish(outputPath) // HTML出力
	if err != nil {
		fmt.Fprintf(os.Stderr, "レポートファイルの出力に失敗しました: %v\n", err)
	}
}
