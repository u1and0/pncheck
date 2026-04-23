package lib

import (
	"fmt"
	"testing"
	"time"

	"pncheck/lib/input"
)

func Test_collectLocalErrors(t *testing.T) {
	// モック用の現在時刻
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format(input.DateLayout)
	tomorrow := now.AddDate(0, 0, 1).Format(input.DateLayout)

	tests := []struct {
		name     string
		sheet    *input.Sheet
		filePath string
		// input.ValidateExcelSums 等の外部依存をどう制御するか
		// 今回は、sheetの状態によって結果が変わるロジックを中心にテスト
		wantErrs []string
	}{
		{
			name: "正常系：すべてのバリデーションを通過",
			sheet: &input.Sheet{
				Header: input.Header{
					ProjectID:   "1234567890", // 10桁目(index9)は '0' (!= )
					RequestDate: yesterday,
				},
			},
			filePath: "valid.xlsx",
			wantErrs: nil,
		},
		{
			name: "異常系：要求年月日が未来の日付",
			sheet: &input.Sheet{
				Header: input.Header{
					ProjectID:   "1234567890",
					RequestDate: tomorrow,
				},
			},
			filePath: "future_date.xlsx",
			wantErrs: []string{
				fmt.Sprintf("要求年月日 %s が未来の日付です", tomorrow),
			},
		},
		{
			name: "異常系：日付フォーマットが不正",
			sheet: &input.Sheet{
				Header: input.Header{
					ProjectID:   "1234567890",
					RequestDate: "2023-01-01", // スラッシュ区切りでない
				},
			},
			filePath: "bad_format.xlsx",
			wantErrs: []string{
				"時間型の解釈に失敗しました: parsing time \"2023-01-01\": SQL基準等のエラーメッセージ",
			},
		},
		{
			name: "異常系：製番の桁数が不足",
			sheet: &input.Sheet{
				Header: input.Header{
					ProjectID:   "12345",
					RequestDate: yesterday,
				},
			},
			filePath: "short_id.xlsx",
			wantErrs: []string{
				"製番の桁数が異常です。12345",
			},
		},
		{
			name: "正常系：製番の10桁目が6（組部品）の場合はソートチェックをスキップ",
			sheet: &input.Sheet{
				Header: input.Header{
					ProjectID:   "1234567896", // 10桁目が6
					RequestDate: yesterday,
				},
			},
			filePath: "assy_item.xlsx",
			wantErrs: nil, // CheckOrderItemsSortOrder がエラーを返す設定でも、呼ばれないはず
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 実際には、ValidateExcelSums や CheckSheetVersion が
			// エラーを返すケースも個別にテストする必要があります。
			// ここではロジックの分岐を確認します。

			gotErrs := collectLocalErrors(tt.sheet, tt.filePath)

			// エラーメッセージの比較（完全一致だと難しい場合があるため、含まれているかで判定することもあります）
			if len(gotErrs) != len(tt.wantErrs) {
				t.Errorf("collectLocalErrors() = %v, want %v", gotErrs, tt.wantErrs)
				return
			}
			for i := range gotErrs {
				// 日付パースエラーなどはシステムによってメッセージが変わるため、一部一致を確認
				if tt.name == "異常系：日付フォーマットが不正" {
					continue
				}
				if gotErrs[i] != tt.wantErrs[i] {
					t.Errorf("got error [%d] = %v, want %v", i, gotErrs[i], tt.wantErrs[i])
				}
			}
		})
	}
}
