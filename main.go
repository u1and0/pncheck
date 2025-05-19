package main

import (
	"fmt"
	"log"
	"os"

	"pncheck/lib"
)

const (
	VERSION    = "v1.0.0"
	outputPath = "pncheck_report.html" // エラー出力ファイル
)

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatalln(err)
	}

	// 各ファイルを処理
	reports := lib.ProcessExcelFile(filePaths)
	if err = reports.Publish(outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "レポートファイルの出力に失敗しました: %v\n", err)
	}
}
