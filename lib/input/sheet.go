package input

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	dateLayout      = "2006/01/02"               // サーバーサイドPNSearchが求める日付の型
	apiEndpointPath = "/api/v1/requests/confirm" // APIのエンドポイントパス
)

var (
	serverAddress  string                                         // APIサーバーのアドレス http://localhost:8080 (ビルド時に注入)
	defaultTimeout = 30 * time.Second                             // API通信のデフォルトタイムアウト
	dateLayoutSub  = []string{"2006/1/2", "1/2/2006", "01-02-06"} // PNSearch規格外の日付文字列
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

func New(fileName string) *Sheet {
	return &Sheet{
		Config: Config{true, true},
		Header: Header{FileName: fileName},
		Orders: make(Orders, 0),
	}
}

// Header.read : 入力II からヘッダー(Header)の読み込み
func (h *Header) read(f *excelize.File) error {
	// 製番 (親番のみ読み取り)
	parentID := getCellValue(f, headerSheetName, projectIDCell)
	edaID := getCellValue(f, headerSheetName, projectEdaCell)
	h.ProjectID = parentID + edaID
	// 製番枝番は読み込まない (必要なら h にフィールド追加し、projectEdaCell から読み込む)
	// h.ProjectEda = getCellValue(f, headerSheetName, projectEdaCell)

	h.ProjectName = getCellValue(f, headerSheetName, projectNameCell)
	// 要求年月日と製番納期は dateLayout の型あるいは空欄に直す
	d := getCellValue(f, headerSheetName, requestDateCell)
	if dd, err := parseDateSafe(d); err != nil {
		return err
	} else {
		h.RequestDate = dd
	}
	d = getCellValue(f, headerSheetName, deadlineHCell)
	if dd, err := parseDateSafe(d); err != nil {
		return err
	} else {
		h.Deadline = dd
	}

	// h.UserSection = getCellValue(f, headerSheetName, userSectionCell)
	h.Note = getCellValue(f, headerSheetName, noteCell)
	return nil
}

func isEmptyRow(pid, name, quantity string) bool {
	return pid == "" && name == "" && quantity == ""
}

func parseIntSafe(s string) (int, error) {
	// ,と.を取り除いて、スペースを削除する
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	s = strings.TrimSpace(strings.ReplaceAll(s, ".", ""))
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func parseFloatSafe(s string) (float64, error) {
	// ,を取り除いて、スペースを削除する
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0.0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// PNSearchが求める日付の文字列型を修正して返す
// パースできないような文字列や
// 空欄はPNSearch側でエラーにならないような処理となっているため
// 影響がないと判断すれば空欄にして返すのが安全
func parseDateSafe(s string) (string, error) {
	switch s { // 棒線や空欄ならそのまま返す
	case "", "‐", "-", "－", "―", "ー":
		return "", nil
	}

	_, err := time.Parse(dateLayout, s)
	if err == nil { // dataLayoutでパースできなければそのまま返す
		return s, nil
	}
	// Excel標準の型でもパースできなければエラー
	for _, l := range dateLayoutSub {
		var t time.Time
		t, err = time.Parse(l, s)
		// fmt.Fprintln(os.Stderr, "[DEBUG]", "parse success", t)
		// パースに成功したらPNSearch標準の文字列型で返す
		if err == nil {
			// fmt.Fprintln(os.Stderr, "[DEBUG]", "return date string", t.Format(dateLayout))
			return t.Format(dateLayout), nil
		}
	}
	return s, err
}

// processOrderRow : 1行分のデータをOrder構造体に変換
func processOrderRow(
	f *excelize.File,
	r int,
	rowPid, rowName, rowQuantityStr string,
) (order Order, err error) {
	lvStr := getCellValue(f, orderSheetName, colLv+strconv.Itoa(r))
	order.Lv, err = parseIntSafe(lvStr)
	if err != nil {
		err = fmt.Errorf("明細(%s) %d行目: Lv(%s)が数値ではありません: %w", orderSheetName, r, colLv, err)
		return
	}

	order.Pid = rowPid
	order.Name = rowName
	order.Type = getCellValue(f, orderSheetName, colType+strconv.Itoa(r))

	// 数量をパース
	order.Quantity, err = parseFloatSafe(rowQuantityStr)
	if err != nil && rowQuantityStr != "" {
		err = fmt.Errorf("明細(%s) %d行目: 数量(%s)が数値ではありません: %w", orderSheetName, r, colQuantity, err)
		return
	}

	order.Unit = getCellValue(f, orderSheetName, colUnit+strconv.Itoa(r))
	// 要望納期は dateLayout の型あるいは空欄に直す
	d := getCellValue(f, orderSheetName, colDeadlineO+strconv.Itoa(r))
	if dd, dateErr := parseDateSafe(d); err != nil {
		err = fmt.Errorf(
			"明細(%s) %d行目: 数量(%s)が正しい日付型%sではありません: %w",
			orderSheetName, r, colDeadlineO, dateLayout, dateErr)
		return
	} else {
		order.Deadline = dd
	}
	order.Kenku = getCellValue(f, orderSheetName, colKenku+strconv.Itoa(r))
	order.Device = getCellValue(f, orderSheetName, colDevice+strconv.Itoa(r))
	order.Serial = getCellValue(f, orderSheetName, colSerial+strconv.Itoa(r))
	order.Maker = getCellValue(f, orderSheetName, colMaker+strconv.Itoa(r))
	order.Vendor = getCellValue(f, orderSheetName, colVendor+strconv.Itoa(r))

	// 予定単価をパース
	unitPriceStr := getCellValue(f, orderSheetName, colUnitPrice+strconv.Itoa(r))
	order.UnitPrice, err = parseFloatSafe(unitPriceStr)
	if err != nil {
		err = fmt.Errorf("明細(%s) %d行目: 予定単価(%s)が数値ではありません: %w", orderSheetName, r, colUnitPrice, err)
		return
	}
	return
}

// read : 入力Ⅰから明細行 (Orders) の読み込み
func (o *Orders) read(f *excelize.File) error {
	emptyRowCount := 0
	for r := ordersStartRow; ; r++ {
		// 1行分のデータを読み込む (主要な列が空かチェック - 品番, 品名, 数量)
		rowPid := getCellValue(f, orderSheetName, colPid+strconv.Itoa(r))
		rowName := getCellValue(f, orderSheetName, colName+strconv.Itoa(r))
		rowQuantityStr := getCellValue(f, orderSheetName, colQuantity+strconv.Itoa(r))

		// 品番、品名、数量がすべて空なら空行とみなす
		if isEmptyRow(rowPid, rowName, rowQuantityStr) {
			emptyRowCount++
			if emptyRowCount >= maxEmptyRowsCheck {
				break // 連続空行が閾値を超えたら終了
			}
			continue // 空行なら次の行へ
		}
		emptyRowCount = 0 // データがあればカウンタリセット

		order, err := processOrderRow(f, r, rowPid, rowName, rowQuantityStr)
		if err != nil {
			return err
		}

		// 読み取ったOrderをスライスに追加
		*o = append(*o, order)
	}
	return nil
}

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

// getCellValue は指定されたセルから値を取得します。エラー時は空文字を返します。
func getCellValue(f *excelize.File, sheetName, axis string) string {
	val, err := f.GetCellValue(sheetName, axis)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(val)
}
