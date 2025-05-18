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


// レスポンスの結果を書き込む
var records []output.ErrorRecord

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
			records = append(records, Record{filePath, "エラーはありません", ""})
			continue
		}

		record := Record{filePath, }
		// JSON型ではないエラーの表示
		// fmt.Fprintf(os.Stderr, "ファイル名:%s, pncheck %s\n", filePath, err)
		err = output.WriteFatal(filePath, err)
		// WriteFatalでもエラーが発生したらFATALLOGに書き込む
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			output.LogFatalError(FATALLOG, fmt.Errorf("エラー: %s への書き込みに失敗しました: %w", FATALLOG, err)
		}
	}
		fmt.Printf("%#v\n", records)
}
