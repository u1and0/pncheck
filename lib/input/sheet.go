package input

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	// サーバーサイドPNSearchが求める日付の型
	dateLayout = "2006/01/02"
	// APIのエンドポイントパス
	apiEndpointPath = "/api/v1/requests/confirm"
)

// 定数定義 (Excelレイアウト - requestパッケージの書き込みコードに基づく)
const (
	headerSheetName = "入力Ⅱ" // ヘッダー情報が主に書かれているシート名
	orderSheetName  = "入力Ⅰ" // 明細情報が書かれているシート名 (requestパッケージの定数を使用)
	versionCell     = "AU1" // 要求票シートのバージョンが書かれているセル

	// --- Header セル位置 (入力Ⅱ) ---
	projectIDCell   = "D1" // 製番 (親番)
	projectEdaCell  = "F1" // 製番 (枝番) - 親番 + 枝番 => 製番　とする
	deadlineHCell   = "D2" // 製番納期
	requestDateCell = "D4" // 要求年月日
	projectNameCell = "D5" // 製番名称
	noteCell        = "D6" // 備考
	userSectionCell = "P5" // 要求元
	// orderTypeCell   = "B2" // 発注区分 (※要確認: 書き込みコードに該当なし、テンプレート依存の可能性大)

	// --- Order セル位置 (入力Ⅰ) ---
	ordersStartRow = 2   // 明細行が始まる行
	colLv          = "A" // Lv列
	colPid         = "E" // 品番列
	colName        = "F" // 品名列
	colType        = "G" // 型式列
	colQuantity    = "I" // 数量列
	colDeadlineO   = "J" // 要望納期列
	colKenku       = "K" // 検区列
	colDevice      = "M" // 装置名列
	colSerial      = "N" // 号機列
	colMaker       = "O" // メーカ列
	// colCompositionQty = "Y" // 構成数量 (固定値1のため読み込み不要)
	colUnit           = "BE" // 単位列
	colVendor         = "BF" // 要望先列
	colUnitPrice      = "BG" // 予定単価列
	maxEmptyRowsCheck = 5    // 連続で何行空行なら明細終了とみなすか
)

var (
	// APIサーバーのアドレス http://localhost:8080 (ビルド時に注入)
	serverAddress string
	// API通信のデフォルトタイムアウト
	defaultTimeout = 30 * time.Second
	// PNSearch規格外の日付文字列
	dateLayoutSub = "01-02-06"
	// 一つだけで十分そう？
	// dateLayoutSub  = []string{"2006/1/2", "1/2/2006", "01-02-06"} // PNSearch規格外の日付文字列
	printSheetName = "10品目用" // 1ページ目の印刷シートの名称
)

type (
	// Config : 設定スイッチ
	Config struct {
		Validatable bool `json:"validatable"` // trueでバリデーション、エラーチェックする
		Sortable    bool `json:"sortable"`    // trueで印刷シートをソートする
		Overridable bool `json:"overridable"` // trueで品名、型式、単位を自動修正する
	}
	// Header : リクエストヘッダー
	Header struct {
		OrderType   OrderType `json:"発注区分"`
		ProjectID   string    `json:"製番"`
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
)

func New(f string) *Sheet {
	return &Sheet{
		Config: Config{true, true, false}, // 初期値はOverridableを実行しない
		Header: Header{
			// ディレクトリを除いたファイル名のみ+surfix _pncheck
			// 30エラーを出さないためのダミーファイル名
			FileName: "pncheck_" + filepath.Base(f),
			// 発注区分をファイル名から分類
			OrderType: parseOrderType(f),
		},
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

	h.Note = getCellValue(f, headerSheetName, noteCell)

	// 要求元は印刷シートから読み込む
	// printSheetName == 10品目用はエラーになり得ないのでエラーを明示的に潰す
	if i, _ := f.GetSheetIndex(printSheetName); i < 0 {
		// 印刷シート名が存在しない(つまりi==1)ならば、入力IIの右隣のシートとする
		// headerSheetName == 入力IIはエラーになり得ないのでエラーを明示的に潰す
		i, _ = f.GetSheetIndex(headerSheetName)
		printSheetName = f.GetSheetName(i + 1)
	}
	h.UserSection = getCellValue(f, printSheetName, userSectionCell)
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
	t, err := time.Parse(dateLayoutSub, s)
	// fmt.Fprintln(os.Stderr, "[DEBUG]", "parse success", t)
	// パースに成功したらPNSearch標準の文字列型で返す
	if err == nil {
		// fmt.Fprintln(os.Stderr, "[DEBUG]", "return date string", t.Format(dateLayout))
		return t.Format(dateLayout), nil
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

// CheckOrderItemsSortOrder : 注文明細の並び順チェック
func CheckOrderItemsSortOrder(sheet Sheet) error {
	orders := sheet.Orders
	n := len(orders)

	// 要素数が0または1の場合は常に正しい並び順とみなす
	if n <= 1 {
		return nil
	}

	// 各要素と次の要素のペアを比較していく
	for i := 0; i < n-1; i++ {
		current := orders[i]
		next := orders[i+1]

		// 比較ルール: 要望納期昇順 -> 品番昇順
		// current が next より後に来ていたら不正

		// 1. 要望納期を比較 (string型での比較)
		// current.Deadline > next.Deadline の場合は不正
		if current.Deadline > next.Deadline {
			return fmt.Errorf(
				"インデックス %d と %d で並べ替え順から外れた項目を注文しています：要望納期 '%s' は '%s' の後です。",
				i, i+1, current.Deadline, next.Deadline,
			)
		}

		// 2. 要望納期が同じ場合、品番を比較 (string型での比較)
		// current.Deadline == next.Deadline かつ current.Pid > next.Pid の場合は不正
		if current.Deadline == next.Deadline {
			if current.Pid > next.Pid {
				return fmt.Errorf("インデックス %d と %d で並べ替え順から外れた項目を注文しています：Pid '%s' は同じ要望納期 '%s' の '%s' の後にあります。",
					i, i+1, current.Pid, next.Pid, current.Deadline)
			}
			// current.Pid <= next.Pid の場合は正しい順序、または同じ要素なのでOK
		}

		// current.Deadline < next.Deadline の場合、または current.Deadline == next.Deadline && current.Pid <= next.Pid の場合は正しい順序、次のペアへ進む
	}

	// 全てのペアの比較が完了し、不正な並び順が見つからなかった
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

// BuildRequestURL : ハッシュ値を基に要求票作成ページを呼び出すためのURLを返す
func BuildRequestURL(sha256 string) string {
	return fmt.Sprintf("%s/index?hash=%s#requirement-tab", serverAddress, sha256)
}

// GetSheetVersion : シート名: “10品目用”のセルAU1 の文字列を返す
func GetSheetVersion(f *excelize.File) string {
	value, err := f.GetCellValue(printSheetName, versionCell)
	if err != nil {
		slog.Error("Get version failed", slog.Any("msg", err.Error()))
		return ""
	}
	return value
}
