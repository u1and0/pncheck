package main

import (
	_ "embed" // embedパッケージをインポート
	"fmt"
	"log"
	"os"
	"time"

	"pncheck/lib"
	"pncheck/lib/input"
)

//go:embed winres/icon.png
var iconContent []byte // icon.pngをバイトスライスとして埋め込む

const (
	VERSION = "v1.6.14"

	outputPath = "pncheck_report.html" // エラー出力ファイル
)

func main() {
	// コマンドライン引数を解析
	filePaths, verboseLevel, err := lib.ParseArguments(VERSION)
	if err != nil {
		log.Fatalln(err)
	}

	// ServerAddress はビルド時 -ldflags で注入される。未設定なら起動時に即終了
	if input.ServerAddress == "" {
		log.Fatalln(
			"APIサーバーアドレスが未設定です。ビルド時に設定する必要があります。\n" +
				`$ go build -ldflags="-X pncheck/lib/input.ServerAddress=http://localhost:8080"`,
		)
	}

	// 各ファイルを処理
	reports, err := lib.ProcessExcelFile(filePaths, verboseLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "レポートファイルの出力に失敗しました: %v\n", err)
	}
	reports.Version = VERSION
	reports.BuildTime = input.BuildTime
	reports.ExecutionTime = time.Now().Format("2006/01/02 15:04:05")
	reports.ServerAddress = input.ServerAddress
	reports.RawIconContent = iconContent // main.goで埋め込んだアイコンコンテンツを渡す

	if verboseLevel > 0 {
		b, err := reports.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSONの標準出力に失敗しました: %v\n", err)
		}
		fmt.Printf("%s\n", string(b)) // 標準出力
	}

	err = reports.Publish(outputPath) // HTML出力
	if err != nil {
		fmt.Fprintf(os.Stderr, "レポートファイルの出力に失敗しました: %v\n", err)
	}
}
