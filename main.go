package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// --- コンパイルを通すための仮定義 ---
type OrderType string // 仮定義

const (
	購入 OrderType = "購入"
	外注 OrderType = "外注"
	出庫 OrderType = "出庫"
	不明 OrderType = "不明" // 不正な区分の場合
)

var db struct { // 仮定義（パッケージ名を模倣）
	ProjectID int // 仮定義
}

// --- ユーザー提供の型定義 ---
type (
	// Config : 設定スイッチ
	Config struct {
		Validatable bool `json:"validatable"` // trueでバリデーション、エラーチェックする
		Sortable    bool `json:"sortable"`    // trueで印刷シートをソートする
	}
	// Header : リクエストヘッダー
	Header struct {
		OrderType   OrderType `json:"発注区分"`
		ProjectID   int       `json:"製番"` // db.ProjectID を int に変更 (仮)
		ProjectName string    `json:"製番名称"`

		RequestDate string `json:"要求年月日"`
		Deadline    string `json:"製番納期"`

		FileName string `json:"ファイル名"`
		// UserSection string `json:"要求元"`
		Note string `json:"備考"`
	}
	// Order : 要求票の1行
	Order struct {
		Lv        int     `json:"Lv"`
		Pid       string  `json:"品番"`
		Name      string  `json:"品名"`
		Type      string  `json:"型式"`
		StockNum  float64 // バックエンド側で在庫数はサーチできるのでPOST不要
		Quantity  float64 `json:"数量"`
		Unit      string  `json:"単位"`
		Deadline  string  `json:"要望納期"`
		Kenku     string  `json:"検区"`
		Device    string  `json:"装置名"`
		Serial    string  `json:"号機"`
		Maker     string  `json:"メーカ"`
		Vendor    string  `json:"要望先"`
		UnitPrice float64 `json:"予定単価"`
		Price     float64 // UnitPriceとQuantityの積なのでPOST不要
	}
	Orders []Order
	// Sheet : JSONでPOSTされる要求票構造体
	Sheet struct {
		Config `json:"config"`
		Header `json:"header"`
		Orders `json:"orders"`
	}

	ErrorRecord struct {
		Message string `json:"msg"`               // 例: "品番が見つかりません"
		Details string `json:"details,omitempty"` // 例: "PN-XXX"
		Key     string `json:"key,omitempty"`     // 例: "品番" or "Pid"
		Index   *int   `json:"index,omitempty"`   // エラーが発生した行番号 (0-based or 1-based? API仕様による)
	}

	PNResponse struct {
		Message string        `json:"msg"`              // ログの概要 (例: "チェック完了", "エラーあり")
		Error   []ErrorRecord `json:"errors,omitempty"` // エラーがあればErrorRecordを追記
		SHA256  string        `json:"sha256,omitempty"` // Sheet構造体から計算したsha256ハッシュ
		Sheet   Sheet         `json:"sheet,omitempty"`  // 検証対象のSheet構造体 (オプション)
	}

	FileProcessResult struct {
		FilePath        string        // 処理対象のファイルパス
		IsSuccess       bool          // プロセス自体が成功したか (ファイル読み込み、API通信、レスポンス解析)
		ValidationError bool          // APIによる検証でエラーが検出されたか (PNResponse.Error が空でない)
		ApiErrors       []ErrorRecord // APIから返されたエラー詳細 (PNResponse.Error)
		ProcessError    error         // プロセス自体のエラー (読み込み、通信、JSON変換など)
	}

	AggregatedResult struct {
		TotalFiles        int                 // 処理した総ファイル数
		SuccessFiles      int                 // プロセス成功ファイル数
		ValidFiles        int                 // プロセス成功 かつ 検証OKファイル数
		InvalidFiles      int                 // プロセス成功 かつ 検証NGファイル数
		ProcessErrorFiles int                 // プロセス失敗ファイル数
		ErrorDetails      []FileProcessResult // エラーがあったファイルの詳細リスト (プロセスエラー or 検証エラー)
	}
)

const (
	defaultErrorFileName = "error_report.log"         // エラーレポートのデフォルトファイル名
	apiEndpointPath      = "/api/v1/requests/confirm" // APIのエンドポイントパス (固定値とする)
	successDirName       = "success"                  // 成功ファイルを移動するディレクトリ名
)

var defaultTimeout = 30 * time.Second // API通信のデフォルトタイムアウト

func main() {
	// 1. コマンドライン引数を解析
	filePaths, err := parseArguments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1) // 引数エラーは終了コード1
	}

	// (任意) ビルド時にサーバーアドレスが設定されているかチェック
	if pnsearchServerAddress == "" {
		fmt.Fprintln(os.Stderr, "エラー: APIサーバーアドレスが設定されていません。")
		fmt.Fprintln(os.Stderr, "       ビルド時に -ldflags=\"-X main.pnsearchServerAddress=http://...\" で指定してください。")
		os.Exit(1)
	}

	// 2. 処理結果を格納するスライスを初期化
	results := make([]FileProcessResult, 0, len(filePaths))

	// 3. 各ファイルを処理
	fmt.Printf("処理を開始します... (%d ファイル)\n", len(filePaths))
	successCount := 0
	errorFound := false // 全体でエラーがあったかを示すフラグ

	// 成功ファイルを移動するディレクトリを作成 (存在してもエラーにならない)
	err = os.MkdirAll(successDirName, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 成功ファイル移動先ディレクトリ '%s' の作成に失敗しました: %v\n", successDirName, err)
		// エラーでも処理は続行する（ファイル移動はスキップされる）
	}

	for i, filePath := range filePaths {
		fmt.Printf("[%d/%d] 処理中: %s ... ", i+1, len(filePaths), filepath.Base(filePath))

		// 3a. 1つのファイルを処理
		result := processExcelFile(filePath)
		results = append(results, result)

		// 3b. 結果表示と成功ファイル移動
		if result.IsSuccess {
			if result.ValidationError {
				fmt.Println("NG (検証エラーあり)")
				errorFound = true
			} else {
				fmt.Println("OK")
				successCount++
				// ファイル移動処理
				if err == nil { // successDirName の作成に成功していれば
					err := moveFileToSuccess(filePath, successDirName)
					if err != nil {
						fmt.Fprintf(os.Stderr, "  警告: ファイル '%s' の移動に失敗しました: %v\n", filepath.Base(filePath), err)
					} else {
						fmt.Printf("  -> '%s' へ移動しました。\n", successDirName)
					}
				}
			}
		} else {
			fmt.Println("エラー (処理失敗)")
			errorFound = true
		}
	}

	fmt.Println("----------------------------------------")
	fmt.Println("処理結果:")

	// 4. 全体の結果を集約
	aggregated := aggregateResults(results)

	// 5. エラーレポートをファイルに出力
	if aggregated.ProcessErrorFiles > 0 || aggregated.InvalidFiles > 0 {
		err = writeErrorFile(aggregated, defaultErrorFileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラーレポートファイル '%s' の書き込みに失敗しました: %v\n", defaultErrorFileName, err)
			errorFound = true // レポート書き込み失敗もエラーとみなす
		} else {
			fmt.Printf("エラーの詳細を '%s' に出力しました。\n", defaultErrorFileName)
		}
	} else {
		fmt.Println("エラーはありませんでした。")
	}

	// 6. サマリー表示
	fmt.Printf("  総ファイル数: %d\n", aggregated.TotalFiles)
	fmt.Printf("  正常処理ファイル数: %d\n", aggregated.SuccessFiles)
	fmt.Printf("    - 検証OK (移動済み): %d\n", successCount) // 移動成功数で表示
	fmt.Printf("    - 検証NG: %d\n", aggregated.InvalidFiles)
	fmt.Printf("  処理エラーファイル数: %d\n", aggregated.ProcessErrorFiles)
	fmt.Println("----------------------------------------")

	// 7. 終了コード設定
	if errorFound {
		fmt.Println("エラーが発生したため、終了コード 1 で終了します。")
		os.Exit(1)
	} else {
		fmt.Println("正常に終了しました。")
		os.Exit(0)
	}
}
