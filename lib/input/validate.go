package input

import (
	"fmt"
	"strconv"
	"time"
)

const (
	projectIDLength  = 12
	projectAssyDigit = 9
	projectAssyValue = 6
)

// CollectLocalErrors はローカルとAPIの一次検証エラーを収集します
func CollectLocalErrors(sheet *Sheet, filePath string) (errs []string) {
	// 各シートの合計値の検証
	if err := ValidateExcelSums(filePath); err != nil {
		errs = append(errs, fmt.Sprintf("合計金額の確認: %s", err))
	}

	// 要求票の版番号
	if err := CheckSheetVersion(sheet.Version); err != nil {
		errs = append(errs, fmt.Sprintf("要求票の版番号の確認: %s", err))
	}

	// 出力日時が要求年月日より未来だったらエラー
	if err := futureRequestValidation(sheet.RequestDate); err != nil {
		errs = append(errs, err.Error())
	}

	// 10桁目が6 == 組部品なのでソートチェックをしない
	if len(sheet.Header.ProjectID) < projectIDLength {
		errs = append(errs, fmt.Sprintf("製番の桁数が異常です。%s", sheet.Header.ProjectID))
	} else {
		class, err := strconv.Atoi(sheet.Header.ProjectID[projectAssyDigit : projectAssyDigit+1])
		if err != nil {
			errs = append(errs, fmt.Sprintf("製番の値が異常です。%s", sheet.Header.ProjectID))
		}
		if class != projectAssyValue {
			// ソートされていることの確認
			if err := sheet.CheckOrderItemsSortOrder(); err != nil {
				errs = append(errs, fmt.Sprintf("入力Iが納期と品番順にソートされていません: %v", err))
			}
		}
	}
	return
}

func futureRequestValidation(reqDate string) error {
	now := time.Now()
	req, err := time.Parse(DateLayout, reqDate)
	if err != nil {
		return fmt.Errorf("時間型の解釈に失敗しました: %w", err.Error())
	}
	if req.After(now) {
		return fmt.Errorf("要求年月日 %s が未来の日付です", reqDate)
	}
	return nil
}
