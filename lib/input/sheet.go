package input

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type OrderType string

const (
	// APIのエンドポイントパス (固定値とする)
	apiEndpointPath           = "/api/v1/requests/confirm"
	購入              OrderType = "購入"
	外注              OrderType = "外注"
	出庫              OrderType = "出庫"
	不明              OrderType = "不明" // 不正な区分の場合
)

var (
	// ValidOrderTypes : ファイル名から判定される発注区分
	ValidOrderTypes = map[string]OrderType{
		"S": 出庫,
		"K": 購入,
		"G": 外注,
	}

	// defaultTimeout : API通信のデフォルトタイムアウト
	defaultTimeout = 30 * time.Second

	// serverAddress : APIサーバーのアドレス
	// "http://localhost:8080" (ビルド時に注入)
	serverAddress string
)

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
)

// Sheet.Post() でサーバーへポスト
// 戻り値はbody, code, error
// code のデフォルト値は500
// errorはPOST自体に失敗した場合のみ設定。
// PostToConfirmAPI は指定されたJSONデータをAPIサーバーにPOSTし、
// レスポンスボディ、HTTPステータスコード、エラーを返します。
// ステータスコードが2xx以外でも、ボディがあれば読み込んで返します。
func (sheet *Sheet) Post() (body []byte, statusCode int, err error) {
	if serverAddress == "" {
		log.Fatalln(
			`APIサーバーアドレスが空です。ビルド時に設定する必要があります。
$ go build -ldflags="-X pncheck/lib/input.serverAddress=http://localhost:8080"`,
		)
	}

	var apiURL = serverAddress + apiEndpointPath
	statusCode = 500 // デフォルト500

	jsonData, err := json.Marshal(sheet)
	if err != nil {
		err = fmt.Errorf("Sheet構造体のJSON変換に失敗しました: %w", err)
		return
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		err = fmt.Errorf("HTTPリクエストの作成に失敗しました: %w", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Do(req)
	if err != nil {
		// 接続エラーなど、レスポンス自体が得られなかった場合
		err = fmt.Errorf("APIへのリクエスト送信に失敗しました (%s): %w", apiURL, err)
		return
	}
	defer resp.Body.Close()

	// レスポンスが得られた場合はステータスコードを記録
	statusCode = resp.StatusCode

	// ボディを読み込む (ステータスコードに関わらず試みる)
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		// ボディ読み込み失敗は致命的エラー
		err = fmt.Errorf("APIレスポンスボディの読み込みに失敗しました: %w", err)
		return
	}
	return
}
