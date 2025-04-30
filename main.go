package main

import (
	"log"

	"pncheck/lib"
)

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments()
	if err != nil {
		log.Fatal(err)
	}

	// 各ファイルを処理
	for _, filePath := range filePaths {
		if err := lib.ProcessExcelFile(filePath); err != nil {
			log.Printf("Error: %s\n", err)
		}
	}
}
