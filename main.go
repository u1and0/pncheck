package main

import (
	"log"

	"pncheck/lib"
	"pncheck/lib/output"
)

const VERSION = "v1.0.0"

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatal(err)
	}

	// 各ファイルを処理
	for _, filePath := range filePaths {
		if err := lib.ProcessExcelFile(filePath); err != nil {
			log.Printf("Error: %s\n", err)
			if err = output.LogFatalError(err); err != nil {
				log.Fatalln("Fatal: ログを記録できません")
			}
		}
	}
}
