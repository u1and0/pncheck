package main

import (
	"log"

	"pncheck/lib"
	"pncheck/lib/output"
)

const (
	VERSION  = "v0.1.0r"
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
		if err == nil {
			continue
		}

		// JSON型ではないエラーの表示
		// fmt.Fprintf(os.Stderr, "ファイル名:%s, pncheck %s\n", filePath, err)
		err = output.WriteFatal(filePath, err)
		// WriteFatalでもエラーが発生したらFATALLOGに書き込む
		if err != nil {
			log.Println(err)
			output.LogFatalError(FATALLOG, err.Error())
		}
	}
}
