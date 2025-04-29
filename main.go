package main

import (
	"fmt"
	"log"
	"os"

	"pncheck/lib"
)

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1) // 引数エラーは終了コード1
	}

	// 各ファイルを処理
	for _, filePath := range filePaths {
		err := lib.ProcessExcelFile(filePath)
		if err != nil {
			log.Printf("Error: %s\n", err)
		}
	}
}
