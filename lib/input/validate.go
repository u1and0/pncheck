package input

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	projectIDLength  = 12
	projectAssyDigit = 9
	projectAssyValue = 6
)

// 合計値を確認するシート名
var sheetsToValidate = []string{
	"入力Ⅱ",
	"印刷用",
	"10品目用",
	"30品目用",
	"100品目用",
}

// CollectLocalErrors はローカルとAPIの一次検証エラーを収集します
func CollectLocalErrors(sheet *Sheet, filePath string) (errs []string) {
	// 各シートの合計値の検証
	if err := validateExcelSums(filePath); err != nil {
		errs = append(errs, fmt.Sprintf("合計金額の確認: %s", err))
	}

	// 要求票の版番号
	if err := checkSheetVersion(sheet.Version); err != nil {
		errs = append(errs, fmt.Sprintf("要求票の版番号の確認: %s", err))
	}

	// 出力日時が要求年月日より未来だったらエラー
	if err := futureRequestValidation(sheet.RequestDate); err != nil {
		errs = append(errs, err.Error())
	}

	// ソート順が異なるとエラー
	if err := sortValidation(sheet); err != nil {
		errs = append(errs, err.Error())
	}

	return
}

func sortValidation(sheet *Sheet) error {
	prjID := sheet.ProjectID
	// 10桁目が6 == 組部品なのでソートチェックをしない
	if len(prjID) < projectIDLength {
		return fmt.Errorf("製番の桁数が異常です。%s", prjID)
	}

	class, err := strconv.Atoi(prjID[projectAssyDigit : projectAssyDigit+1])
	if err != nil {
		return fmt.Errorf("製番の値が異常です。%s", prjID)
	}
	// 組部品はソートされてなくてOK
	if class == projectAssyValue {
		return nil
	}

	// ソートされていることの確認
	if err := checkOrderItemsSortOrder(sheet.Orders); err != nil {
		return fmt.Errorf("入力Iが納期と品番順にソートされていません: %v", err)
	}

	return nil
}

// checkOrderItemsSortOrder : 注文明細の並び順チェック
func checkOrderItemsSortOrder(orders Orders) error {
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
		if current.Deadline == next.Deadline && current.Pid > next.Pid {
			return fmt.Errorf("インデックス %d と %d で並べ替え順から外れた項目を注文しています：品番 '%s' は同じ要望納期 '%s' の '%s' の後にあります。",
				i, i+1, current.Pid, next.Pid, current.Deadline)
		}
		// current.Pid <= next.Pid の場合は正しい順序、または同じ要素なのでOK

		// current.Deadline < next.Deadline の場合、または current.Deadline == next.Deadline && current.Pid <= next.Pid の場合は正しい順序、次のペアへ進む
	}

	// 全てのペアの比較が完了し、不正な並び順が見つからなかった
	return nil
}

// validateExcelSums はExcelシート内の合計値が正しいか検証します。
func validateExcelSums(filePath string) error {
	opts := excelize.Options{RawCellValue: true}
	f, err := excelize.OpenFile(filePath, opts)
	if err != nil {
		slog.Warn("ファイルを開けません。合計値チェックをスキップします。",
			slog.String("filePath", filePath),
			slog.String("error", err.Error()),
		)
		return nil
	}
	defer f.Close()

	for _, sheetName := range sheetsToValidate {
		i, err := f.GetSheetIndex(sheetName)
		if err != nil || i < 0 {
			slog.Warn(fmt.Sprintf("シート '%s' が見つかりません。スキップします。", sheetName), slog.String("sheet", sheetName))
			continue
		}

		// レンジの合計値算出
		config, err := getSheetValidationConfig(f, sheetName)
		if err != nil {
			return fmt.Errorf("%sシートの合計計算設定エラー: %w", sheetName, err)
		}
		sum, err := sumCellRange(f, sheetName, config.cellRange)
		if err != nil {
			return fmt.Errorf("%sシートの合計計算エラー: %w", sheetName, err)
		}

		// レンジの合計と上下それぞれの合計値が等しくなければエラーを返す
		err = fmt.Errorf(
			"%sシートにおいて、%s の合計が正しく計算できていません",
			sheetName, config.cellRange,
		)
		valUpperSumCell := getFloatCellValue(f, sheetName, config.upperSumCell)
		if sum != valUpperSumCell {
			return err
		}
		valCellSum := getFloatCellValue(f, sheetName, config.cellSum)
		if sum != valCellSum {
			return err
		}
	}

	return nil
}

// checkSheetVersion : 要求票の版番号確認を行う
// sheet.Header.Version は開いているExcelファイルから読み取ったシートのバージョンです。
// この関数は、サーバーから最新のシートバージョンを取得し、sheet.Header.Version と比較します。
// バージョンが一致しない場合、エラーを返します。
//
// 要求票の版番号の確認はサーバーへ GETメソッド
// http://192.168.160.118:9000/api/v1/requests/version
//
// 想定されるレスポンス:
// {"sheetVersion":"M-0-814-04"}
func checkSheetVersion(localVersion string) error {
	// バージョンが空文字列の場合の警告（サーバー側またはローカル側）
	if localVersion == "" {
		slog.Warn("ローカルシートのバージョンが空です。サーバーと比較できません。")
	}

	// サーバーテンプレートのバージョンを取得
	if ServerAddress == "" {
		slog.Warn("APIサーバーアドレスが未設定のため、バージョンチェックをスキップします。",
			slog.String("hint", `go build -ldflags="-X pncheck/lib/input.ServerAddress=http://localhost:8080"`),
		)
		return nil
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

	if serverSheetVersion == "" {
		slog.Warn("サーバーからのシートバージョンが空です。比較に失敗しました。", slog.
			String("apiURL", apiURL))
		// サーバーのバージョンが空の場合、有効なバージョンではないとみなしエラーを返す
		return fmt.Errorf("サーバーから有効なシートバージョンが取得できませんでした。")
	}

	// バージョンの比較
	if localVersion != serverSheetVersion {
		return fmt.Errorf(
			"要求票のバージョンが一致しません。"+
				"ローカル: '%s', サーバー: %s' です。"+
				"最新の要求票テンプレートをご利用ください。",
			localVersion, serverSheetVersion,
		)
	}
	return nil
}

// futureRequestValidation : 出力日時が要求年月日より未来だったらエラー
func futureRequestValidation(reqDate string) error {
	now := time.Now()
	req, err := time.Parse(DateLayout, reqDate)
	if err != nil {
		return fmt.Errorf("時間型の解釈に失敗しました: %w", err)
	}
	if req.After(now) {
		return fmt.Errorf("要求年月日 %s が未来の日付です", reqDate)
	}
	return nil

}
