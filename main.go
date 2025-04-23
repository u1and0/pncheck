package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

// --- コンパイルを通すための仮定義 ---
type OrderType string

const (
	購入 OrderType = "購入"
	外注 OrderType = "外注"
	出庫 OrderType = "出庫"
	不明 OrderType = "不明" // 不正な区分の場合
)

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
		ProjectID   string    `json:"製番"`
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

	// サーバーからのエラー出力のうち、このコマンドで使用するもの
	ErrorOutput struct {
		Filename string        `json:"filename"`
		Msg      string        `json:"msg"`
		Errors   []ErrorRecord `json:"errors,omitempty"`
	}
)

const (
	errorReportFileName = "error_report_log.json"    // エラーレポートのデフォルトファイル名
	apiEndpointPath     = "/api/v1/requests/confirm" // APIのエンドポイントパス (固定値とする)
	successDirName      = "success"                  // 成功ファイルを移動するディレクトリ名
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

	// 2. 変数初期化
	var results []string

	// success ディレクトリ作成 (変更なし)
	err = os.MkdirAll(successDirName, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: success ディレクトリ作成失敗: %v\n", err)
	}

	// 3. 各ファイルを処理
	// エラーがあったらerror_report.jsonへ追記する
	// エラーがなければsuccessディレクトリへ移動
	for _, filePath := range filePaths {
		b, err := processExcelFile(filePath)
		fmt.Println(string(b))
		if err != nil {
			err = writeErrorFile(b, errorReportFileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			continue
		} else {
			err = moveFileToSuccess(filePath, successDirName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		}
		log.Printf("response JSON: %v\n", results)
	}
}
