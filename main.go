package main

import (
	"fmt"
	"log"
	"os"

	"pncheck/lib"
	"pncheck/lib/output"
)

const (
	VERSION  = "v0.1.0"
	FATALLOG = "pncheck_fatal_report.json"
)

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatal(err)
	}

	// 各ファイルを処理
	for _, filePath := range filePaths {
		if err := lib.ProcessExcelFile(filePath); err != nil {
			// エラーがあったら標準エラーに出力した後FATALLOGに書き込む
			fmt.Fprintf(os.Stderr, "PNCheck Error: %s\n", err)
			if err = output.LogFatalError(FATALLOG, err); err != nil {
				log.Fatalf("Fatal: %s にログを記録できません\n", FATALLOG)
			}
		}
	}
}
