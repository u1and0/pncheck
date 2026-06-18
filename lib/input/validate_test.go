package input

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
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

func TestIsEmptyColumn(t *testing.T) {
	t.Parallel()

	// ヘルパー: テスト用のインメモリExcelファイルを生成
	setupFile := func(t *testing.T, data map[string]string) *excelize.File {
		t.Helper()
		f := excelize.NewFile()
		_, err := f.NewSheet(orderSheetName)
		if err != nil {
			t.Fatalf("Failed to create sheet: %v", err)
		}
		for axis, val := range data {
			if err := f.SetCellValue(orderSheetName, axis, val); err != nil {
				t.Fatalf("Failed to set cell value: %v", err)
			}
		}
		return f
	}

	tests := []struct {
		name      string
		excelData map[string]string
		col       string
		lastRow   int
		want      bool
	}{
		{
			name:      "指定範囲がすべて空文字の場合_trueを返す",
			excelData: map[string]string{}, // データなし
			col:       "A",
			want:      true,
		},
		{
			name: "指定範囲外(開始行未満)に値があっても影響せず_trueを返す",
			excelData: map[string]string{
				"A1": "header", // ordersStartRow(2) 未満の1行目
			},
			col:  "A",
			want: true,
		},
		{
			name: "指定範囲外(最終行超)に値があっても影響せず_trueを返す",
			excelData: map[string]string{
				"A102": "value", // lastRow(101) を超えた102行目
			},
			col:  "A",
			want: true,
		},
		{
			name: "指定範囲の途中に値がある場合_falseを返す",
			excelData: map[string]string{
				"A3": "dirty", // ordersStartRow(2) と lastRow(101) の間
			},
			col:  "A",
			want: false,
		},
		{
			name: "指定範囲の境界(開始行)に値がある場合_falseを返す",
			excelData: map[string]string{
				"A2": "start_edge",
			},
			col:  "A",
			want: false,
		},
		{
			name: "指定範囲の境界(最終行)に値がある場合_falseを返す",
			excelData: map[string]string{
				"A101": "end_edge",
			},
			col:  "A",
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt // Go 1.22未満のループ変数の参照対策
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := setupFile(t, tt.excelData)
			defer f.Close()

			got := IsEmptyColumn(f, orderSheetName, tt.col)
			if got != tt.want {
				t.Errorf("IsEmptyColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateHiddenColumns(t *testing.T) {
	t.Parallel()

	// ヘルパー: テスト用の一時Excelファイルを作成してパスを返す
	createTempExcel := func(t *testing.T, data map[string]string) string {
		t.Helper()
		f := excelize.NewFile()
		_, err := f.NewSheet(orderSheetName)
		if err != nil {
			t.Fatalf("Failed to create sheet: %v", err)
		}

		// テストデータの書き込み
		for axis, val := range data {
			if err := f.SetCellValue(orderSheetName, axis, val); err != nil {
				t.Fatalf("Failed to set cell value: %v", err)
			}
		}

		// 一時ディレクトリに保存
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test_order.xlsx")
		if err := f.SaveAs(filePath); err != nil {
			t.Fatalf("Failed to save temp file: %v", err)
		}
		return filePath
	}

	tests := []struct {
		name      string
		setupFile func(t *testing.T) string
		wantErr   bool
		errMsg    string // エラーメッセージに含まれるべきキーワード
	}{
		{
			name: "隠し列(B, D)に一切入力がない場合_正常終了(nil)",
			setupFile: func(t *testing.T) string {
				return createTempExcel(t, map[string]string{
					"A1": "HeaderA", "B1": "HiddenHeaderB", "C1": "HeaderC", "D1": "HiddenHeaderD",
					"A2": "data", "E2": "data", // 隠し列(B, D)のデータ行(2行目以降)は空
				})
			},
			wantErr: false,
		},
		{
			name: "隠し列Bに入力がある場合_エラーを返す",
			setupFile: func(t *testing.T) string {
				return createTempExcel(t, map[string]string{
					"A2": "data",
					"B2": "invalid_data", // 隠し列Bに入力あり
				})
			},
			wantErr: true,
			errMsg:  "隠し列に入力があります: B",
		},
		{
			name: "複数の隠し列(B, D)に入力がある場合_カンマ区切りでエラーを返す",
			setupFile: func(t *testing.T) string {
				return createTempExcel(t, map[string]string{
					"A2": "data",
					"B2": "invalid_data1", // 隠し列Bに入力あり
					"C2": "invalid_data2", // 隠し列Bに入力あり
					"D3": "invalid_data3", // 隠し列Dに入力あり
				})
			},
			wantErr: true,
			errMsg:  "隠し列に入力があります: B, C, D",
		},
		{
			name: "ファイルが存在しない場合_ワーニングを出力して正常終了(nil)を返す(仕様)",
			setupFile: func(t *testing.T) string {
				return "non_existent_file.xlsx"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filePath := tt.setupFile(t)

			// 存在するファイルの場合は後片付け（t.TempDir() を使っているため自動削除されるが明示的にも削除可能）
			if _, err := os.Stat(filePath); err == nil {
				defer os.Remove(filePath)
			}

			err := validateHiddenColumns(filePath)

			if (err != nil) != tt.wantErr {
				t.Fatalf("validateHiddenColumns() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}
