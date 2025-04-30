package main

import (
	"log"
	"os"

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
		// 渡されたファイルがディレクトリの場合は無視
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Fatal(err)
		}
		if fileInfo.IsDir() {
			log.Printf("Error: %s is directory \n", filePath)
			continue
		}
		err = lib.ProcessExcelFile(filePath)
		if err != nil {
			log.Printf("Error: %s\n", err)
		}
	}
}
