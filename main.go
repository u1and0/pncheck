package main

import (
	"fmt"
	"log"
	"os"

	"pncheck/lib"
)

const (
	VERSION = "v0.1.0r"
)

func main() {
	// コマンドライン引数を解析
	filePaths, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatal(err)
	}

	// 各ファイルを処理
	for _, filePath := range filePaths {
		err := lib.ProcessExcelFile(filePath)
		if err == nil {
			continue
		}
		if err = lib.WriteError(filePath, err); err != nil {
	if err = output.LogFatalError(FATALLOG, msg); err != nil {
		log.Fatalf("Fatal: %s にログを記録できません\n", FATALLOG)
	}
		}
	}
}
