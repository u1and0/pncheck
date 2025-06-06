package lib

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath" // ヘルプメッセージ用にインポート
)

// ParseArguments はコマンドライン引数を解析し、処理対象のExcelファイルパスのリストを返します。
// 引数が指定されていない場合や、-h / --help が指定された場合はヘルプメッセージを表示して終了します。
func ParseArguments(version string) (filePaths []string, err error) {
	// ヘルプフラグの定義
	var showHelp bool
	flag.BoolVar(&showHelp, "h", false, "ヘルプメッセージを表示します")
	flag.BoolVar(&showHelp, "help", false, "ヘルプメッセージを表示します")

	// バージョンフラグの定義
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "バージョン情報を表示します")
	flag.BoolVar(&showVersion, "version", false, "バージョン情報を表示します")

	// 使用法メッセージのカスタマイズ
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "指定されたExcelファイルをPNSearch APIでチェックします。\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [オプション] <Excelファイルパス1> [Excelファイルパス2] ...\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults() // 定義されたフラグの説明を表示
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s request1.xlsx request2.xlsx\n", filepath.Base(os.Args[0]))
	}

	flag.Parse() // コマンドライン引数をパース

	// ヘルプフラグが指定されたらUsageを表示して終了(成功)
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// バージョンフラグが指定されたらバージョンを表示して終了(成功)
	if showVersion {
		fmt.Println(filepath.Base(os.Args[0]), version)
		os.Exit(0)
	}

	// フラグ以外の引数（ファイルパス）を取得
	filePaths = flag.Args()

	// ファイルパスが1つも指定されていない場合はエラー
	if len(filePaths) == 0 {
		flag.Usage() // 使い方も表示
		return nil, errors.New("処理対象のExcelファイルを最低1つ指定してください")
	}

	// ここで各ファイルパスの存在チェックや拡張子チェックを行うことも可能だが、
	// processExcelFile内でエラーハンドリングするため、ここでは必須としない。

	return filePaths, nil
}
