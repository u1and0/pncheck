package main

import "time"

// --- コンパイルを通すための仮定義 ---
// 実際には適切なパッケージをimportしてください
// errorsは必要なのでimport

type OrderType string // 仮定義

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

		FileName    string `json:"ファイル名"`
		UserSection string `json:"要求元"`
		Note        string `json:"備考"`
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

const defaultErrorFileName = "error_report.log" // エラーレポートのデフォルトファイル名
const apiEndpointPath = "/confirm"              // APIのエンドポイントパス (固定値とする)
var defaultTimeout = 30 * time.Second           // API通信のデフォルトタイムアウト
