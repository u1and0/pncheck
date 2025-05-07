package main

import (
	"fmt"
	"log"
	"os"

	"pncheck/lib"
	"pncheck/lib/output"
)

const (
	VERSION  = "v0.1.1"
	FATALLOG = "pncheck_fatal_report.log"
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
			msg := fmt.Sprintf("ファイル名:%s, pncheck %s\n", filePath, err)
			fmt.Fprintf(os.Stderr, msg)
			if err = output.LogFatalError(FATALLOG, msg); err != nil {
				log.Fatalf("Fatal: %s にログを記録できません\n", FATALLOG)
			}
		}
	}
}
