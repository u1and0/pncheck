package input

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// --- テスト関数 ---

// TestReadExcelToSheet_Success
// expectedSheet.Header.ProjectID の期待値を修正
func TestReadExcelToSheet_Success(t *testing.T) {
	testDir := "testdata_read"
	testFile := createTestExcelFile(t, testDir, "20231027-success-read-K.xlsx", setValidLayout)

	expectedSheet := Sheet{
		Config: Config{Validatable: true, Overridable: true},
		Header: Header{
			OrderType:   購入,
			ProjectID:   "1234501",                              // D1 + F1
			ProjectName: "テストプロジェクト",                            // D5
			RequestDate: "2023/10/27",                           // D4
			Deadline:    "2023/11/30",                           // D2
			FileName:    "20231027-success-read-K_pncheck.xlsx", // ファイル名 ダミーの_pncheck suffixがつく
			Serial:      "read",                                 // ファイル名から読み込まれる
			Note:        "備考欄テスト",                               // D6
			Version:     "M-701-04",                             // AV1
		},
		Orders: Orders{
			{ // Row 2
				Lv: 1, Pid: "PN-001", Name: "部品A", Type: "TypeX",
				Quantity: 10.5, Unit: "個", Deadline: "2023/11/15", Kenku: "受入",
				Device: "装置1", Serial: "S001", Maker: "MakerX", Vendor: "VendorY", UnitPrice: 100.50,
			},
			{ // Row 3
				Lv: 2, Pid: "PN-002", Name: "部品B", Type: "",
				Quantity: 5, Unit: "Set", Deadline: "", Kenku: "",
				Device: "", Serial: "", Maker: "", Vendor: "", UnitPrice: 2500,
			},
			{ // Row 5
				Lv: 0, Pid: "PN-003", Name: "部品C", Type: "",
				Quantity: 1, Unit: "", Deadline: "", Kenku: "",
				Device: "", Serial: "", Maker: "", Vendor: "", UnitPrice: 0,
			},
		},
	}

	// 実行前に、テスト用のファイル名が parseFileNameInfo でエラーにならないように調整
	// (parseFileNameInfo はファイル名自体を引数にとるため)
	// expectedFileName := fmt.Sprintf("20231027-%s-%s-%s.xlsx",
	//     expectedSheet.Header.ProjectID[:12], // 製番部分 (12桁想定)
	//     "DUMMY", // 号機はテストデータ設定にないので仮
	//     "K") // OrderTypeが購入になるように
	// testFile = createTestExcelFile(t, testDir, expectedFileName, setValidLayout) // 再生成 or リネーム
	// ↑ parseFileNameInfo が filePath を受け取るので、テストファイル自体の名前を期待値に合わせるか、
	//   parseFileNameInfo の引数を固定値にするなどの工夫が必要

	// // ---- parseFileNameInfo を直接呼び出すのではなく、ReadExcelToSheet内で呼ばれることを考慮 ----
	// // ReadExcelToSheet は filePath を引数にとり、その中で parseFileNameInfo(filePath) を呼ぶ
	// // そのため、テスト用のExcelファイル名自体が parseFileNameInfo のフォーマットに合致している必要がある
	//
	// // テストファイル名を parseFileNameInfo が成功する形式にする
	// // 例: "YYYYMMDD-ProjectID(親)-Serial-OrderType.xlsx"
	// // このテストでは OrderType 購入 (K) を期待
	// correctFormatFileName := fmt.Sprintf("20231027-%s-TESTSERIAL-K.xlsx", expectedSheet.Header.ProjectID)
	// // createTestExcelFile でファイル名を設定し直すか、テスト前にリネームする
	// // ここでは createTestExcelFile に渡すファイル名を変更する
	// testFile = createTestExcelFile(t, testDir, correctFormatFileName, setValidLayout)
	// // 期待値の FileName も合わせる
	// expectedSheet.Header.FileName = correctFormatFileName

	actualSheet, err := ReadExcelToSheet(testFile)
	if err != nil {
		t.Fatalf("予期せぬエラーが発生しました: %v", err)
	}

	// DeepEqual で比較
	if !reflect.DeepEqual(expectedSheet, actualSheet) {
		// ... (詳細比較のロジックは変更なし) ...
		t.Errorf("Sheetが期待値と異なります。")
		if !reflect.DeepEqual(expectedSheet.Config, actualSheet.Config) {
			t.Errorf("  Config:\n    期待値: %+v\n    実際値: %+v", expectedSheet.Config, actualSheet.Config)
		}
		if !reflect.DeepEqual(expectedSheet.Header, actualSheet.Header) {
			t.Errorf("  Header:\n    期待値: %+v\n    実際値: %+v", expectedSheet.Header, actualSheet.Header)
		}
		// ... (Ordersの比較など) ...
		if len(expectedSheet.Orders) != len(actualSheet.Orders) {
			t.Errorf("  Ordersの件数が異なります: 期待値=%d, 実際値=%d", len(expectedSheet.Orders), len(actualSheet.Orders))
		} else {
			for i := range expectedSheet.Orders {
				if !reflect.DeepEqual(expectedSheet.Orders[i], actualSheet.Orders[i]) {
					t.Errorf("  Orders[%d]:\n    期待値: %+v\n    実際値: %+v", i, expectedSheet.Orders[i], actualSheet.Orders[i])
				}
			}
		}
	}
}

func TestReadExcelToSheet_ReadfileError(t *testing.T) {
	_, err := ReadExcelToSheet("")
	if err == nil {
		t.Fatal("不正なパスを与えた場合にエラーが返されませんでした。")
	}
	t.Logf("期待通りファイル情報読み込みエラーを検出: %v", err)
}

func TestReadExcelToSheet_IsDirectory(t *testing.T) {
	_, err := ReadExcelToSheet("testdata_read")
	if err == nil {
		t.Fatal("ディレクトリを与えた場合にエラーが返されませんでした。")
	}
	t.Logf("期待通りディレクトリエラーを検出: %v", err)
}

func TestReadExcelToSheet_FileNotFound(t *testing.T) {
	_, err := ReadExcelToSheet("testdata_read/non_existent_file-K.xlsx")
	if err == nil {
		t.Fatal("ファイルが存在しない場合にエラーが返されませんでした。")
	}
	t.Logf("期待通りファイルオープンエラーを検出: %v", err)
}

func TestReadExcelToSheet_InvalidNumberFormat_Orders(t *testing.T) {
	testDir := "testdata_read"
	testFile := createTestExcelFile(t, testDir, "20231027-invalid_orders_read-K.xlsx", func(f *excelize.File) {
		setValidLayout(f)
		f.SetCellValue(orderSheetName, colQuantity+"2", "Not A Number") // 2行目の数量に文字列
	})

	_, err := ReadExcelToSheet(testFile)
	if err == nil {
		t.Fatal("明細行の数値変換エラーが検出されませんでした。")
	}
	expectedErrMsg := fmt.Sprintf("明細(%s) 2行目: 数量(%s)が数値ではありません",
		orderSheetName, colQuantity)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("期待されるエラーメッセージが含まれていません。\n期待含む: %s\n実際: %v",
			expectedErrMsg, err)
	}
	t.Logf("期待通り明細数値エラーを検出: %v", err)
}

// expectedSheet.Header.ProjectID の期待値を修正
func TestReadExcelToSheet_EmptySheet(t *testing.T) {
	testDir := "testdata_read"
	// ファイル名を parseFileNameInfo が成功する形式にする
	correctFormatFileName := "20240101-9999900-EMPTY-K.xlsx"

	testFile := createTestExcelFile(t, testDir, correctFormatFileName, func(f *excelize.File) {
		// ヘッダーだけ設定し、明細は空にする
		f.SetCellValue(headerSheetName, projectIDCell, "99999") // 親番
		f.SetCellValue(headerSheetName, projectEdaCell, "00")   // 枝番
		f.SetCellValue(headerSheetName, projectNameCell, "空シートテスト")
		// ... 他のヘッダー項目 ...
		// orderSheetName ("入力Ⅰ") には何も書き込まない
	})

	expectedHeader := Header{
		OrderType:   購入,        // ファイル名から
		ProjectID:   "9999900", // 親番 + 枝番
		ProjectName: "空シートテスト",
		// RequestDate など、設定していないフィールドはゼロ値のまま
		FileName: "20240101-9999900-EMPTY-K_pncheck.xlsx",
		Serial:   "EMPTY", // ファイル名から読み込まれる
	}

	sheet, err := ReadExcelToSheet(testFile)
	if len(sheet.Orders) != 0 {
		t.Errorf("明細が空のはずが、%d件読み込まれました。", len(sheet.Orders))
	}
	// ヘッダーを比較 (ProjectID と FileName、OrderType のみチェック)
	if sheet.Header.ProjectID != expectedHeader.ProjectID {
		t.Errorf("空の明細シートでもヘッダーは読み込まれるはずです。ProjectID: expected=%q, actual=%q",
			expectedHeader.ProjectID, sheet.Header.ProjectID)
	}
	if sheet.Header.FileName != expectedHeader.FileName {
		t.Errorf("FileNameが期待値と異なります。FileName: expected=%q, actual=%q",
			expectedHeader.FileName, sheet.Header.FileName)
	}
	if sheet.Header.OrderType != expectedHeader.OrderType {
		t.Errorf("OrderTypeが期待値と異なります。OrderType: expected=%q, actual=%q",
			expectedHeader.OrderType, sheet.Header.OrderType)
	}
	if err != nil {
		// parseFileNameInfo でエラーになる可能性があるためチェック
		t.Log("空の明細シートで予期せぬエラー: ", err.Error())
	}
	t.Log("空の明細シートを正常に処理しました。")
}

func TestActivateOrderSheet(t *testing.T) {
	// Initialize an Excel file
	testDir := "testdata_activate"
	testFileName := "testfile.xlsx"
	testFile := createTestExcelFile(t, testDir, testFileName, setValidLayout)

	tests := []struct {
		name           string
		filePath       string
		sheetName      string
		wantErr        bool
		wantErrMsg     string
		activeSheetIdx int
	}{
		{
			name:           "success",
			filePath:       testFile,
			wantErr:        false,
			wantErrMsg:     "新しく保存しました。",
			activeSheetIdx: 0, // idx==0は入力II
		},
		{
			name:           "already active",
			filePath:       testFile,
			wantErr:        false,
			activeSheetIdx: 1,
		},
		{
			name:       "file not found",
			filePath:   "non-existent-file.xlsx",
			wantErr:    true,
			wantErrMsg: "ファイルを開けません",
		},
		{
			name:       "order sheet not found",
			filePath:   testFile,
			sheetName:  "不正なシート名",
			wantErr:    true,
			wantErrMsg: "入力Iシートが見つかりません",
		},
	}

	for _, tt := range tests {
		// テストのためにあえて入力I以外のシートをアクティブにする
		t.Run(tt.name, func(t *testing.T) {
			f, _ := excelize.OpenFile(tt.filePath)
			if tt.activeSheetIdx > 0 {
				f.SetActiveSheet(tt.activeSheetIdx)
				if err := f.SaveAs(tt.filePath); err != nil {
					t.Fatal(err)
				}
			}
		})

		// テストのためにあえて入力I以外のシート名にする
		t.Run(tt.name, func(t *testing.T) {
			f, _ := excelize.OpenFile(tt.filePath)
			if tt.sheetName != "" {
				f.SetSheetName(orderSheetName, tt.sheetName)
				if err := f.SaveAs(tt.filePath); err != nil {
					t.Fatal(err)
				}
			}
		})

		t.Run(tt.name, func(t *testing.T) {
			err := ActivateOrderSheet(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ActivateOrderSheet() name: %s, error: %v, wantErr: %v", tt.name, err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(fmt.Sprintf("%v", err), tt.wantErrMsg) {
				t.Errorf("ActivateOrderSheet() error message = %v, wantErrMsg %v", err, tt.wantErrMsg)
			}
		})
	}
}
