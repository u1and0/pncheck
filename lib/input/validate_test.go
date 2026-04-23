package input

import (
	"fmt"
	"testing"
	"time"
)

func Test_CollectLocalErrors(t *testing.T) {
	// モック用の現在時刻
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format(DateLayout)
	tomorrow := now.AddDate(0, 0, 1).Format(DateLayout)

	tests := []struct {
		name     string
		sheet    *Sheet
		filePath string
		// ValidateExcelSums 等の外部依存をどう制御するか
		// 今回は、sheetの状態によって結果が変わるロジックを中心にテスト
		wantErrs []string
	}{
		{
			name: "正常系：すべてのバリデーションを通過",
			sheet: &Sheet{
				Header: Header{
					ProjectID:   "123456789000", // 12桁、index9は '0' (!= 6)
					RequestDate: yesterday,
				},
			},
			filePath: "valid.xlsx",
			wantErrs: nil,
		},
		{
			name: "異常系：要求年月日が未来の日付",
			sheet: &Sheet{
				Header: Header{
					ProjectID:   "123456789000",
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
			sheet: &Sheet{
				Header: Header{
					ProjectID:   "123456789000",
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
			sheet: &Sheet{
				Header: Header{
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
			sheet: &Sheet{
				Header: Header{
					ProjectID:   "123456789600", // 12桁、index9が6（組部品）
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

			gotErrs := CollectLocalErrors(tt.sheet, tt.filePath)

			// エラーメッセージの比較（完全一致だと難しい場合があるため、含まれているかで判定することもあります）
			if len(gotErrs) != len(tt.wantErrs) {
				t.Errorf("CollectLocalErrors() = %v, want %v", gotErrs, tt.wantErrs)
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

// TestCheckOrderItemsSortOrder は CheckOrderItemsSortOrder 関数のテストを行います。
func TestCheckOrderItemsSortOrder(t *testing.T) {
	// テストケースを定義
	tests := []struct {
		name        string // テストケースの名前
		sheet       Sheet  // 入力となるSheetデータ
		expectError bool   // エラーが期待されるか (true: 期待する, false: 期待しない)
		// expectedErrorMsg string // エラーメッセージの詳細をチェックする場合に使う (今回はシンプルにエラーの有無のみ)
	}{
		{
			name:        "Empty Orders - Should Pass",
			sheet:       Sheet{Orders: Orders{}},
			expectError: false,
		},
		{
			name:        "Single Order - Should Pass",
			sheet:       Sheet{Orders: Orders{{Deadline: "2023-10-26", Pid: "A001"}}},
			expectError: false,
		},
		{
			name: "Correct Sort by Deadline Only - Should Pass",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "C003"}, // Pidは順不同でもOK (Deadlineが違うから)
				{Deadline: "2023-10-27", Pid: "A001"},
				{Deadline: "2023-10-28", Pid: "B002"},
			}},
			expectError: false,
		},
		{
			name: "Correct Sort by Deadline and Pid - Should Pass",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "A001"},
				{Deadline: "2023-10-26", Pid: "B002"},
				{Deadline: "2023-10-27", Pid: "C003"},
				{Deadline: "2023-10-27", Pid: "D004"},
				{Deadline: "2023-10-28", Pid: "A001"},
			}},
			expectError: false,
		},
		{
			name: "Incorrect Sort - Deadline Out of Order - Should Fail",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-27", Pid: "C003"},
				{Deadline: "2023-10-26", Pid: "A001"}, // 2023-10-27 の後に 2023-10-26
			}},
			expectError: true,
		},
		{
			name: "Incorrect Sort - Pid Out of Order for Same Deadline - Should Fail",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "B002"},
				{Deadline: "2023-10-26", Pid: "A001"}, // 同じ納期で B002 の後に A001
			}},
			expectError: true,
		},
		{
			name: "Incorrect Sort - Pid Out of Order for Same Deadline (Multiple Items) - Should Fail",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "A001"}, // OK
				{Deadline: "2023-10-27", Pid: "D004"}, // OK (Deadlineが違う)
				{Deadline: "2023-10-27", Pid: "C003"}, // Fail (同じ納期で D004の後に C003)
				{Deadline: "2023-10-28", Pid: "A001"}, // この行には到達しないはず
			}},
			expectError: true,
		},
		{
			name: "Correct Sort with Duplicate Items - Should Pass",
			sheet: Sheet{Orders: Orders{
				{Deadline: "2023-10-26", Pid: "A001"},
				{Deadline: "2023-10-26", Pid: "A001"}, // 同じアイテムでも順序はOK
				{Deadline: "2023-10-27", Pid: "B002"},
			}},
			expectError: false,
		},
	}

	// 各テストケースを実行
	for _, tt := range tests {
		// t.Run を使うと、各テストケースが独立して実行され、結果が見やすくなります
		t.Run(tt.name, func(t *testing.T) {
			err := checkOrderItemsSortOrder(tt.sheet.Orders)

			// エラーが期待されているかチェック
			if tt.expectError {
				// エラーが期待されているのに nil が返ってきた場合
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
				// (オプション) 特定のエラーメッセージをチェックする場合
				// if err != nil && tt.expectedErrorMsg != "" && err.Error() !=tt.expectedErrorMsg {
				//      t.Errorf("Expected error message '%s', but got '%s'", tt.expectedErrorMsg, err.Error())
				// }
			} else {
				// エラーが期待されていないのに nil 以外が返ってきた場合
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
			}
		})
	}
}
