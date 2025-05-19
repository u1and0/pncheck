package main

import (
	"fmt"
	"log"
	"os"

	"pncheck/lib"
	"pncheck/lib/output"
)

const (
	VERSION  = "v0.1.1r"
	FATALLOG = "pncheck_fatal_report.log"
)

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatalln(err)
		output.LogFatalError(FATALLOG, err.Error())
	}

	// 各ファイルを処理
	for _, filePath := range filePaths {
		err := lib.ProcessExcelFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ファイル名:%s, pncheck %s\n", filePath, err)
			continue
		}
		// 成功したらコンソールに成功メッセージを書くだけ
		fmt.Fprintf(os.Stderr, "ファイル名:%s エラーはありませんでした。\n", filePath)
	}
}
