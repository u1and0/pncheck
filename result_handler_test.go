package main

import (
	"errors"
	"reflect"
	"testing"
)

// --- aggregateResults のテスト ---

func TestAggregateResults(t *testing.T) {
	index1 := 0
	index5 := 5

	testResults := []FileProcessResult{
		// 1. プロセス成功、検証OK
		{FilePath: "file1.xlsx", IsSuccess: true, ValidationError: false},
		// 2. プロセス成功、検証NG (APIエラーあり)
		{FilePath: "file2.xlsx", IsSuccess: true, ValidationError: true, ApiErrors: []ErrorRecord{
			{Message: "必須項目エラー", Key: "Pid"},
			{Message: "フォーマットエラー", Key: "Quantity", Index: &index5},
		}},
		// 3. プロセス失敗 (ファイル読み込みエラー)
		{FilePath: "file3.xlsx", IsSuccess: false, ProcessError: errors.New("ファイルオープンエラー")},
		// 4. プロセス成功、検証OK
		{FilePath: "file4.xlsx", IsSuccess: true, ValidationError: false},
		// 5. プロセス成功、検証NG (APIエラーあり)
		{FilePath: "file5.xlsx", IsSuccess: true, ValidationError: true, ApiErrors: []ErrorRecord{
			{Message: "未登録", Key: "Maker", Details: "MakerX", Index: &index1},
		}},
		// 6. プロセス失敗 (API通信エラー)
		{FilePath: "file6.xlsx", IsSuccess: false, ProcessError: errors.New("APIタイムアウト")},
	}

	expected := AggregatedResult{
		TotalFiles:        6,
		SuccessFiles:      4, // file1, file2, file4, file5
		ValidFiles:        2, // file1, file4
		InvalidFiles:      2, // file2, file5
		ProcessErrorFiles: 2, // file3, file6
		ErrorDetails: []FileProcessResult{ // エラーがあったもの (検証NG または プロセスNG)
			testResults[1], // file2 (検証NG)
			testResults[2], // file3 (プロセスNG)
			testResults[4], // file5 (検証NG)
			testResults[5], // file6 (プロセスNG)
		},
	}

	actual := aggregateResults(testResults)

	// 各カウンターを比較
	if actual.TotalFiles != expected.TotalFiles {
		t.Errorf("TotalFiles: 期待=%d, 実際=%d", expected.TotalFiles, actual.TotalFiles)
	}
	if actual.SuccessFiles != expected.SuccessFiles {
		t.Errorf("SuccessFiles: 期待=%d, 実際=%d", expected.SuccessFiles, actual.SuccessFiles)
	}
	if actual.ValidFiles != expected.ValidFiles {
		t.Errorf("ValidFiles: 期待=%d, 実際=%d", expected.ValidFiles, actual.ValidFiles)
	}
	if actual.InvalidFiles != expected.InvalidFiles {
		t.Errorf("InvalidFiles: 期待=%d, 実際=%d", expected.InvalidFiles, actual.InvalidFiles)
	}
	if actual.ProcessErrorFiles != expected.ProcessErrorFiles {
		t.Errorf("ProcessErrorFiles: 期待=%d, 実際=%d", expected.ProcessErrorFiles, actual.ProcessErrorFiles)
	}

	// ErrorDetails の内容を比較 (件数とファイルパスで簡易的に)
	if len(actual.ErrorDetails) != len(expected.ErrorDetails) {
		t.Errorf("ErrorDetailsの件数: 期待=%d, 実際=%d", len(expected.ErrorDetails), len(actual.ErrorDetails))
	} else {
		// ファイルパスが一致するか確認 (順番も同じはず)
		for i := range expected.ErrorDetails {
			if actual.ErrorDetails[i].FilePath != expected.ErrorDetails[i].FilePath {
				t.Errorf("ErrorDetails[%d] のファイルパスが異なります: 期待=%s, 実際=%s", i, expected.ErrorDetails[i].FilePath, actual.ErrorDetails[i].FilePath)
			}
			// 必要であれば、エラー内容なども詳細に比較する
		}
	}
}

func TestAggregateResults_EmptyInput(t *testing.T) {
	actual := aggregateResults([]FileProcessResult{})
	expected := AggregatedResult{
		TotalFiles:        0,
		SuccessFiles:      0,
		ValidFiles:        0,
		InvalidFiles:      0,
		ProcessErrorFiles: 0,
		ErrorDetails:      []FileProcessResult{},
	}
	if !reflect.DeepEqual(actual, expected) { // 構造体は直接比較可能
		t.Errorf("空の入力に対する結果が期待値と異なります。\n期待: %+v\n実際: %+v", expected, actual)
	}
}

// --- writeErrorFile のテスト ---

// func TestWriteErrorFile_Success(t *testing.T) {
// 	testDir := "testdata_writer"
// 	outputFile := filepath.Join(testDir, "report.log")
// 	index2 := 2
//
// 	aggregated := AggregatedResult{
// 		TotalFiles:        3,
// 		SuccessFiles:      1,
// 		InvalidFiles:      1,
// 		ProcessErrorFiles: 1,
// 		ErrorDetails: []FileProcessResult{
// 			{FilePath: "path/to/error1.xlsx", IsSuccess: true, ValidationError: true, ApiErrors: []ErrorRecord{
// 				{Message: "値が不正です", Key: "UnitPrice", Details: "abc"}, // Indexなし
// 				{Message: "必須項目です", Key: "Pid", Index: &index2},
// 			}},
// 			{FilePath: "another/error2.xlsx", IsSuccess: false, ProcessError: errors.New("ファイル読み込み失敗\tタブあり\n改行あり")},
// 		},
// 	}
//
// 	// ディレクトリ作成
// 	if err := os.MkdirAll(testDir, 0755); err != nil {
// 		t.Fatalf("テストディレクトリ作成失敗: %v", err)
// 	}
// 	// テスト終了時にファイルとディレクトリ削除
// 	t.Cleanup(func() {
// 		os.Remove(outputFile)
// 		os.Remove(testDir)
// 	})
//
// 	err := writeErrorFile(aggregated, outputFile)
// 	if err != nil {
// 		t.Fatalf("writeErrorFile で予期せぬエラー: %v", err)
// 	}
//
// 	// ファイルが作成されたか確認
// 	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
// 		t.Fatalf("エラーファイル '%s' が作成されませんでした", outputFile)
// 	}
//
// 	// ファイルの内容を確認
// 	contentBytes, err := os.ReadFile(outputFile)
// 	if err != nil {
// 		t.Fatalf("エラーファイルの読み込みに失敗: %v", err)
// 	}
// 	content := string(contentBytes)
//
// 	// 期待される内容 (ヘッダー + 3行のエラー詳細)
// 	expectedLines := []string{
// 		"ファイルパス\tエラー種別\tエラー内容\tエラー箇所キー\tエラー箇所詳細\tエラー行",
// 		"path/to/error1.xlsx\t検証エラー\t値が不正です\tUnitPrice\tabc\t-",
// 		"path/to/error1.xlsx\t検証エラー\t必須項目です\tPid\t-\t2",
// 		"another/error2.xlsx\tプロセスエラー\tファイル読み込み失敗\\tタブあり\\n改行あり\t-\t-\t-", // タブと改行がエスケープされていること
// 	}
// 	actualLines := strings.Split(strings.TrimSpace(content), "\n") // 末尾の改行を削除してから分割
//
// 	if len(actualLines) != len(expectedLines) {
// 		t.Errorf("エラーファイルの行数が異なります: 期待=%d, 実際=%d", len(expectedLines), len(actualLines))
// 		t.Logf("実際の内容:\n%s", content)
// 	} else {
// 		for i := range expectedLines {
// 			// Windows環境での改行コードの違い(CRLF)を考慮して TrimSpace する
// 			if strings.TrimSpace(actualLines[i]) != strings.TrimSpace(expectedLines[i]) {
// 				t.Errorf("エラーファイルの %d 行目の内容が異なります:\n期待: %s\n実際: %s", i+1, expectedLines[i], actualLines[i])
// 			}
// 		}
// 	}
// }
//
// func TestWriteErrorFile_NoErrors(t *testing.T) {
// 	testDir := "testdata_writer_noerr"
// 	outputFile := filepath.Join(testDir, "no_report.log")
//
// 	aggregated := AggregatedResult{
// 		TotalFiles:   2,
// 		SuccessFiles: 2,
// 		ValidFiles:   2,
// 		ErrorDetails: []FileProcessResult{}, // エラーなし
// 	}
//
// 	// ディレクトリ作成 (後で削除)
// 	if err := os.MkdirAll(testDir, 0755); err != nil {
// 		t.Fatalf("テストディレクトリ作成失敗: %v", err)
// 	}
// 	t.Cleanup(func() { os.Remove(outputFile); os.Remove(testDir) })
//
// 	err := writeErrorFile(aggregated, outputFile)
// 	if err != nil {
// 		t.Fatalf("エラーがない場合に予期せぬエラー: %v", err)
// 	}
//
// 	// ファイルが作成されていないことを確認
// 	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
// 		t.Errorf("エラーがない場合にファイル '%s' が作成されてしまいました", outputFile)
// 	}
// }
//
// func TestWriteErrorFile_EmptyPath(t *testing.T) {
// 	aggregated := AggregatedResult{
// 		ErrorDetails: []FileProcessResult{{FilePath: "dummy.xlsx", IsSuccess: false, ProcessError: errors.New("dummy")}}, // エラーはある
// 	}
// 	err := writeErrorFile(aggregated, "") // 空のパス
// 	if err == nil {
// 		t.Fatal("出力パスが空の場合にエラーが返されませんでした")
// 	}
// 	if !strings.Contains(err.Error(), "出力パスが指定されていません") {
// 		t.Errorf("期待されるエラーメッセージではありません: %v", err)
// 	}
// }
//
// func TestWriteErrorFile_CantCreateFile(t *testing.T) {
// 	// 書き込み権限のないディレクトリを指定 (テスト環境によっては難しい場合あり)
// 	// または、ディレクトリではなく既存のファイルを指定する
// 	nonWritablePath := "/dev/null/cannot_write_here.log" // 例 (Unix系)
// 	// Windowsの場合: "C:/Windows/cannot_write_here.log" など (管理者権限が必要な場所)
// 	// もっと確実なのは、テスト用に一時ファイルを作成し、それをディレクトリとして扱おうとするパスにする
// 	tmpFile, err := os.CreateTemp("", "readonly_")
// 	if err != nil {
// 		t.Fatalf("一時ファイル作成失敗: %v", err)
// 	}
// 	tmpFilePath := tmpFile.Name()
// 	tmpFile.Close()
// 	defer os.Remove(tmpFilePath)
//
// 	nonWritablePath = filepath.Join(tmpFilePath, "report.log") // 既存ファイルの下にファイルを作ろうとする
//
// 	aggregated := AggregatedResult{
// 		ErrorDetails: []FileProcessResult{{FilePath: "dummy.xlsx", IsSuccess: false, ProcessError: errors.New("dummy")}},
// 	}
// 	err = writeErrorFile(aggregated, nonWritablePath)
// 	if err == nil {
// 		t.Fatal("ファイル作成不可なパスでエラーが返されませんでした")
// 	}
// 	if !strings.Contains(err.Error(), "作成に失敗しました") {
// 		t.Errorf("期待されるエラーメッセージではありません: %v", err)
// 	}
// 	t.Logf("期待通りファイル作成失敗エラーを検出: %v", err)
// }
