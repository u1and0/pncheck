package input

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"net/http"
	"path/filepath"
	"regexp"
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
	// サーバーの要求票バージョン取得AP
	apiVersionEndpointPath = "/api/v1/requests/version"
)

// 定数定義 (Excelレイアウト - requestパッケージの書き込みコードに基づく)
const (
	headerSheetName       = "入力Ⅱ"   // ヘッダー情報が主に書かれているシート名
	orderSheetName        = "入力Ⅰ"   // 明細情報が書かれているシート名 (requestパッケージの定数を使用)
	printSheetNameDefault = "10品目用" // 1ページ目の印刷シートの名称

	// --- Header セル位置 (入力Ⅱ) ---
	projectIDCell   = "D1"  // 製番 (親番)
	projectEdaCell  = "F1"  // 製番 (枝番) - 親番 + 枝番 => 製番　とする
	deadlineHCell   = "D2"  // 製番納期
	requestDateCell = "D4"  // 要求年月日
	projectNameCell = "D5"  // 製番名称
	noteCell        = "D6"  // 備考
	userSectionCell = "Q5"  // 要求元
	versionCell     = "AV1" // 要求票シートのバージョンが書かれているセル
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
	colMisc           = "AJ"  // 備考列
	colUnit           = "BE"  // 単位列
	colVendor         = "BF"  // 要望先列
	colUnitPrice      = "BG"  // 予定単価列
	remarkCell        = "AJ3" // 備考(組部品用 出庫指示番号)
	maxEmptyRowsCheck = 5     // 連続で何行空行なら明細終了とみなすか
)

var (
	// API通信のデフォルトタイムアウト
	defaultTimeout = 30 * time.Second
	// PNSearch規格外の日付文字列
	dateLayoutSub = []string{"01-02-06", "2006/1/2", "1/2/2006"} // PNSearch規格外の日付文字列
)

type (
	// Config : 設定スイッチ
	Config struct {
		Validatable bool `json:"validatable"` // trueでバリデーション、エラーチェックする
		Overridable bool `json:"overridable"` // trueで品名、型式、単位を自動修正する
		Mergeable   bool `json:"mergeable"`   // trueで組部品非登録用シートを一つにまとめる
	}
	// Header : リクエストヘッダー
	Header struct {
		OrderType   OrderType `json:"発注区分"`
		ProjectID   string    `json:"製番"`
		ProjectName string    `json:"製番名称"`

		RequestDate string `json:"要求年月日"`
		Deadline    string `json:"製番納期"`
		Remark      string `json:"出庫指示番号(組部品用)"`

		FileName    string `json:"ファイル名"`
		Serial      string `json:"号機"`
		UserSection string `json:"要求元"`
		Note        string `json:"備考"`

		Version string
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

	// ServerVersionResponse はサーバーから返されるバージョン情報のJSON構造を定義します。
	ServerVersionResponse struct {
		SheetVersion string `json:"sheetVersion"`
	}
)

// New はファイルパスfからシート構造の初期値を出力する。
func New(f string) *Sheet {
	return &Sheet{
		Config: Config{true, true, true},
		Header: Header{
			FileName:  newFileName(f),    // _pncheckを付与
			OrderType: parseOrderType(f), // 発注区分をファイル名から分類
			Serial:    parseSerial(f),    // 号機をファイル名から取得
		},
		Orders: make(Orders, 0),
	}
}

// newFileName はディレクトリを除いたファイル名のみ+surfix _pncheck
// ファイル名重複エラーを出さないためのダミーファイル名として、
// 末尾にpncheckをつける
func newFileName(f string) string {
	base, ext := filepath.Base(f), filepath.Ext(f)
	return strings.TrimSuffix(base, ext) + "_pncheck" + ext
}

// parseSerial はファイルパスからSerialを読み込む
func parseSerial(f string) string {
	base, ext := filepath.Base(f), filepath.Ext(f)
	noext := strings.TrimSuffix(base, ext)
	fields := strings.Split(noext, "-")
	if len(fields) < 3 {
		return ""
	}
	return fields[2]
}

// Header.read : 入力II からヘッダー(Header)の読み込み
func (h *Header) read(f *excelize.File) error {
	// 製番 (親番のみ読み取り)
	parentID := getCellValue(f, headerSheetName, projectIDCell)
	edaID := getCellValue(f, headerSheetName, projectEdaCell)
	h.ProjectID = strings.TrimSpace(parentID) + strings.TrimSpace(edaID)
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

	// getDispatchNumber 備考欄の出庫指示番号は入力Iから読み込む
	h.Remark = getLastRemarkValue(f)

	// 印刷シート名の取得
	printSheetName := getPrintSheet(f)

	// シートの版番号の取得
	ver, err := f.GetCellValue(printSheetName, versionCell)
	localSheetVersion := strings.TrimSpace(ver)
	if err != nil || localSheetVersion == "" {
		return fmt.Errorf(
			"要求票ファイルからバージョン情報を読み取れませんでした。"+
				"セル'%s' が空か存在しない可能性があります。",
			versionCell,
		)
	}
	h.Version = localSheetVersion

	// 要求元は印刷シートから読み込む
	h.UserSection = getCellValue(f, printSheetName, userSectionCell)

	// 備考(組部品用 出庫指示番号)は入力1から読み込む
	// "出庫指示番号33690による"のような文字列が入る
	s := getCellValue(f, orderSheetName, remarkCell)
	// この中から数値だけ抜き出して、string型で取り出す処理
	re := regexp.MustCompile(`\d+`)
	h.Remark = re.FindString(s)
	return nil
}

// getPrintSheet : 10品目用という名前のシートが見つからなければ
// 入力IIの右隣にのシート名とする
func getPrintSheet(f *excelize.File) string {
	// printSheetName == 10品目用はエラーになり得ないのでエラーを明示的に潰す
	if i, _ := f.GetSheetIndex(printSheetNameDefault); i < 0 {
		// 印刷シート名が存在しない(つまりi==1)ならば、入力IIの右隣のシートとする
		// headerSheetName == 入力IIはエラーになり得ないのでエラーを明示的に潰す
		i, _ = f.GetSheetIndex(headerSheetName)
		return f.GetSheetName(i + 1)
	}
	return printSheetNameDefault
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
//
// まずはdateLayout, dateLayoutSub に定めた文字列型として解釈し、
// 失敗したらExcel日付型として解釈する。
//
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
	for _, layoutSub := range dateLayoutSub {
		var t time.Time
		t, err = time.Parse(layoutSub, s)
		// fmt.Fprintln(os.Stderr, "[DEBUG]", "parse success", t)
		// パースに成功したらPNSearch標準の文字列型で返す

		// DEBUG
		// fmt.Fprintln(os.Stderr,
		// 	fmt.Sprintf("[DEBUG]%sで%sをParseした結果: %s",
		// 		layoutSub, s, t))
		if err == nil {
			return t.Format(dateLayout), nil
		}

	}

	// 文字列型で読み込めなければExcelTime(1900年1月1日が0)
	if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
		t := excelTimeToGoTime(v)
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
func (sheet *Sheet) CheckOrderItemsSortOrder() error {
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
				return fmt.Errorf("インデックス %d と %d で並べ替え順から外れた項目を注文しています：品番 '%s' は同じ要望納期 '%s' の '%s' の後にあります。",
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
	if ServerAddress == "" {
		log.Fatalln(
			`APIサーバーアドレスが空です。ビルド時に設定する必要があります。
$ go build -ldflags="-X pncheck/lib/input.ServerAddress=http://localhost:8080"`,
		)
	}

	var apiURL = ServerAddress + apiEndpointPath
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
	s, err := f.GetCellValue(sheetName, axis)
	if err != nil {
		return ""
	}
	// 左右のスペース、タブ文字、改行削除
	s = strings.TrimSpace(s)
	// 中間の改行削除
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\r", " ")
}

// getLastRemarkValue finds the last non-empty row in column AJ (備考欄) and extracts
// the dispatch number from it.
func getLastRemarkValue(f *excelize.File) string {
	// Find the last non-empty row in column AJ (備考欄)
	lastRow := ordersStartRow - 1 // Start from the row before the first data row
	for r := ordersStartRow; ; r++ {
		// Check if the main columns (品番, 品名, 数量) are all empty
		rowPid := getCellValue(f, orderSheetName, colPid+strconv.Itoa(r))
		rowName := getCellValue(f, orderSheetName, colName+strconv.Itoa(r))
		rowQuantityStr := getCellValue(f, orderSheetName, colQuantity+strconv.Itoa(r))

		if isEmptyRow(rowPid, rowName, rowQuantityStr) {
			break // Stop when we find an empty row
		}
		lastRow = r // Update lastRow to current row
	}

	// Read the remark from the last non-empty row in column AJ
	remark := getCellValue(f, orderSheetName, colMisc+strconv.Itoa(lastRow))
	re := regexp.MustCompile(`\d+`) // 正規表現で数値のみ抜き出し
	return re.FindString(remark)
}

// getFloatCellValue は指定されたセルから値をfloat64型で取得します。エラー時は0.0を返します。
func getFloatCellValue(f *excelize.File, sheetName, axis string) float64 {
	s := getCellValue(f, sheetName, axis)
	val, err := parseFloatSafe(s)
	if err != nil {
		slog.Warn(
			"セル値の数値変換に失敗しました。",
			slog.String("sheet", sheetName),
			slog.String("cell", axis),
			slog.String("value", s),
			slog.String("error", err.Error()),
		)
		return 0.0
	}
	return val
}

// sumCellRange は指定されたシートとセル範囲の値を合計します。
// 例: "A1:A10"
func sumCellRange(f *excelize.File, sheetName, cellRange string) (float64, error) {
	total := 0.0
	// セル範囲をパース (例: "A1:A10")
	parts := strings.Split(cellRange, ":")
	if len(parts) != 2 {
		return 0.0, fmt.Errorf("無効なセル範囲形式: %s", cellRange)
	}

	startColInt, startRowInt, err := excelize.CellNameToCoordinates(parts[0])
	if err != nil {
		return 0.0, fmt.Errorf("開始セル参照のパースに失敗しました: %w", err)
	}
	endColInt, endRowInt, err := excelize.CellNameToCoordinates(parts[1])
	if err != nil {
		return 0.0, fmt.Errorf("終了セル参照のパースに失敗しました: %w", err)
	}

	startCol := startColInt
	startRow := startRowInt
	endCol := endColInt
	endRow := endRowInt

	// 現在の要件では単一列の範囲のみを想定しているため、列が異なる場合はエラーとする
	if startCol != endCol {
		return 0.0, fmt.Errorf("複数列にまたがる範囲の合計はサポートしていません: %s", cellRange)
	}

	for r := startRow; r <= endRow; r++ {
		cellAxis, err := excelize.CoordinatesToCellName(startCol, r)
		if err != nil {
			slog.Warn(
				"セル座標から名前への変換に失敗しました。",
				slog.Int("col", startCol),
				slog.Int("row", r),
				slog.String("error", err.Error()),
			)
			continue
		}
		total += getFloatCellValue(f, sheetName, cellAxis)
	}
	return total, nil
}

// BuildRequestURL : ハッシュ値を基に要求票作成ページを呼び出すためのURLを返す
func BuildRequestURL(sha256 string) string {
	return fmt.Sprintf("%s/index?hash=%s#requirement-tab", ServerAddress, sha256)
}

// CheckSheetVersion : 要求票の版番号確認を行う
// sheet.Header.Version は開いているExcelファイルから読み取ったシートのバージョンです。
// この関数は、サーバーから最新のシートバージョンを取得し、sheet.Header.Version と比較します。
// バージョンが一致しない場合、エラーを返します。
//
// 要求票の版番号の確認はサーバーへ GETメソッド
// http://192.168.160.118:9000/api/v1/requests/version
//
// 想定されるレスポンス:
// {"sheetVersion":"M-0-814-04"}
func (sheet *Sheet) CheckSheetVersion() error {
	// サーバーテンプレートのバージョンを取得
	if ServerAddress == "" {
		// ビルド時に ServerAddress が設定されていない場合は致命的エラー
		log.Fatalln(
			`APIサーバーアドレスが空です。ビルド時に設定する必要があります。
$ go build -ldflags="-X pncheck/lib/input.ServerAddress=http://localhost:8080"`,
		)
	}

	apiURL := ServerAddress + apiVersionEndpointPath
	client := &http.Client{Timeout: defaultTimeout}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("HTTPリクエストの作成に失敗しました (%s): %w", apiURL, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("APIへのリクエスト送信に失敗しました (%s): %w", apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // エラーボディも読み込んでログに含める
		return fmt.Errorf(
			"サーバーからのバージョン取得に失敗しました。ステータスコード: %d, レスポンス: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("APIレスポンスボディの読み込みに失敗しました: %w", err)
	}

	var serverResp ServerVersionResponse
	if err := json.Unmarshal(body, &serverResp); err != nil {
		return fmt.Errorf("サーバー応答のJSON解析に失敗しました: %w, レスポンス: %s",
			err, string(body))
	}

	serverSheetVersion := serverResp.SheetVersion

	// バージョンが空文字列の場合の警告（サーバー側またはローカル側）
	if sheet.Header.Version == "" {
		slog.Warn("ローカルシートのバージョンが空です。サーバーと比較できません。")
	}
	if serverSheetVersion == "" {
		slog.Warn("サーバーからのシートバージョンが空です。比較に失敗しました。", slog.
			String("apiURL", apiURL))
		// サーバーのバージョンが空の場合、有効なバージョンではないとみなしエラーを返す
		return fmt.Errorf("サーバーから有効なシートバージョンが取得できませんでした。")
	}

	// バージョンの比較
	if sheet.Header.Version != serverSheetVersion {
		return fmt.Errorf(
			"要求票のバージョンが一致しません。"+
				"ローカル: '%s', サーバー: %s' です。"+
				"最新の要求票テンプレートをご利用ください。",
			sheet.Header.Version, serverSheetVersion,
		)
	}
	return nil
}

// Excelのシリアル値をGoの time.Time に変換するヘルパー関数
// Excel Time型は整数部分が日数を、小数部分が時刻を表す
func excelTimeToGoTime(excelSerialValue float64) time.Time {
	// 1900年ベースの場合の起点は 1900/1/1 (シリアル値 1)
	excelSerialValue -= 2.0
	// 日数部分の計算
	days := math.Floor(excelSerialValue)
	// 秒数の計算 (24時間 * 3600秒/時間 = 86400秒/日)
	seconds := math.Round((excelSerialValue - days) * 86400.0)

	// 基準日 (1900年1月1日)
	// Goの time.Time は1900年1月1日 00:00:00 (JST) から日数を加算する
	baseTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

	// 基準日から日数と秒数を加算して返す
	t := baseTime.AddDate(0, 0, int(days)).Add(time.Duration(seconds) * time.Second)

	return t
}
