package main

import (
	"encoding/json"
	"fmt"
	"io" // エラー比較用に import
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// --- テスト用データ ---
var testValidSheet = Sheet{ // convertToJSON, postToConfirmAPI のテストで使用
	Config: Config{Validatable: true, Sortable: false},
	Header: Header{
		ProjectID:   "000001234512345",
		ProjectName: "テスト",
		FileName:    "test.xlsx",
	},
	Orders: Orders{
		{Pid: "PN-001", Quantity: 10, UnitPrice: 100},
		{Pid: "PN-002", Quantity: 5, UnitPrice: 200},
	},
}

// --- convertToJSON のテスト ---

func TestConvertToJSON_Success(t *testing.T) {
	sheet := testValidSheet
	jsonData, err := convertToJSON(sheet)
	if err != nil {
		t.Fatalf("予期せぬエラー: %v", err)
	}

	// 簡単なJSON内容チェック (特定のキーが存在するかなど)
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("生成されたJSONのパースに失敗: %v", err)
	}

	if _, ok := result["config"]; !ok {
		t.Error("JSONに 'config' キーが含まれていません")
	}
	if _, ok := result["header"]; !ok {
		t.Error("JSONに 'header' キーが含まれていません")
	}
	if _, ok := result["orders"]; !ok {
		t.Error("JSONに 'orders' キーが含まれていません")
	}

	// Header内のキーをチェック
	headerMap, ok := result["header"].(map[string]interface{})
	if !ok {
		t.Fatal("JSONの 'header' がマップではありません")
	}
	if _, ok := headerMap["製番"]; !ok { // json:"製番" タグを確認
		t.Error("JSONの 'header' に '製番' キーが含まれていません")
	}

	// Ordersが配列かチェック
	ordersArray, ok := result["orders"].([]interface{})
	if !ok {
		t.Fatal("JSONの 'orders' が配列ではありません")
	}
	if len(ordersArray) != len(sheet.Orders) {
		t.Errorf("JSONの 'orders' の要素数が異なります: 期待値=%d, 実際値=%d", len(sheet.Orders), len(ordersArray))
	}
	// Orders内のキーをチェック (最初の要素)
	if len(ordersArray) > 0 {
		orderMap, ok := ordersArray[0].(map[string]interface{})
		if !ok {
			t.Fatal("JSONの 'orders' の要素がマップではありません")
		}
		if _, ok := orderMap["品番"]; !ok { // json:"品番" タグを確認
			t.Error("JSONの 'orders' の要素に '品番' キーが含まれていません")
		}
		if _, ok := orderMap["数量"]; !ok {
			t.Error("JSONの 'orders' の要素に '数量' キーが含まれていません")
		}
	}
}

func TestConvertToJSON_EmptySheet(t *testing.T) {
	sheet := Sheet{} // 空の構造体
	jsonData, err := convertToJSON(sheet)
	if err != nil {
		t.Fatalf("予期せぬエラー: %v", err)
	}

	// 空のJSON (`{"config":{},"header":{},"orders":null}` のような形になるはず)
	expectedJSON := `{"config":{"validatable":false,"sortable":false},"header":{"発注区分":"","製番":"","製番名称":"","要求年月日":"","製番納期":"","ファイル名":"","備考":""},"orders":null}`
	// reflect.DeepEqual だと orders:null と orders:[] で差が出る可能性があるので文字列比較
	if string(jsonData) != expectedJSON {
		t.Errorf("空のSheetのJSONが期待値と異なります。\n期待値: %s\n実際値: %s", expectedJSON, string(jsonData))
	}

	// 一応パース可能か確認
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("生成された空JSONのパースに失敗: %v", err)
	}
}

// --- postToConfirmAPI のテスト ---

func TestPostToConfirmAPI_Success(t *testing.T) {
	// モックサーバーの準備
	expectedResponse := `{"msg":"チェックOK","sha256":"abcde"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// リクエストメソッドとパスを検証
		if r.Method != "POST" {
			t.Errorf("期待しないメソッド: %s", r.Method)
		}
		if r.URL.Path != apiEndpointPath {
			t.Errorf("期待しないパス: %s", r.URL.Path)
		}
		// Content-Typeを検証
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("期待しないContent-Type: %s", r.Header.Get("Content-Type"))
		}
		// リクエストボディを検証 (オプション)
		bodyBytes, _ := io.ReadAll(r.Body)
		var receivedSheet Sheet
		if err := json.Unmarshal(bodyBytes, &receivedSheet); err != nil {
			t.Errorf("リクエストボディのJSONパース失敗: %v", err)
		}
		// ここで receivedSheet の内容を testValidSheet と比較しても良い

		// 正常レスポンスを返す
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, expectedResponse)
	}))
	defer server.Close()

	// テスト対象関数を実行
	jsonData, _ := convertToJSON(testValidSheet)
	responseBody, err := postToConfirmAPI(jsonData, server.URL) // モックサーバーのアドレスを使用

	// 結果を検証
	if err != nil {
		t.Fatalf("予期せぬエラー: %v", err)
	}
	actualResponseBody := strings.TrimSpace(string(responseBody))
	if string(actualResponseBody) != expectedResponse {
		t.Errorf("レスポンスボディが期待値と異なります。\n期待値: %s\n実際値: %s (Trim後: %s)", expectedResponse, string(responseBody), actualResponseBody)
	}
}

func TestPostToConfirmAPI_ApiError(t *testing.T) {
	// モックサーバー (400 Bad Request を返す)
	errorMsg := `{"msg":"リクエスト形式エラー"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json") // エラーでもJSONを返す想定
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, errorMsg)
	}))
	defer server.Close()

	// テスト対象関数を実行
	jsonData, _ := convertToJSON(testValidSheet)
	_, err := postToConfirmAPI(jsonData, server.URL)

	// 結果を検証 (エラーが発生し、メッセージにステータスコードとレスポンスが含まれること)
	if err == nil {
		t.Fatal("APIエラー時にエラーが返されませんでした")
	}
	if !strings.Contains(err.Error(), "ステータス: 400") {
		t.Errorf("エラーメッセージにステータスコード(400)が含まれていません: %v", err)
	}
	if !strings.Contains(err.Error(), errorMsg) {
		t.Errorf("エラーメッセージにAPIレスポンスが含まれていません: %v", err)
	}
	t.Logf("期待通りAPIエラーを検出: %v", err)
}

func TestPostToConfirmAPI_Timeout(t *testing.T) {
	// モックサーバー (指定時間以上待機してから応答)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // defaultTimeoutより短い時間にする
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"msg":"OK"}`)
	}))
	defer server.Close()

	// タイムアウト時間を短く設定したクライアントでテスト (本番コードに影響を与えずにテスト)
	originalTimeout := defaultTimeout
	timeoutForTest := 10 * time.Millisecond             // サーバーのSleepより短く設定
	defaultTimeout = timeoutForTest                     // グローバル変数を一時的に変更 (テスト中は注意)
	defer func() { defaultTimeout = originalTimeout }() // テスト終了後に元に戻す

	// テスト対象関数を実行
	jsonData, _ := convertToJSON(testValidSheet)
	_, err := postToConfirmAPI(jsonData, server.URL)

	// 結果を検証 (タイムアウトエラーが発生すること)
	if err == nil {
		t.Fatal("タイムアウト時にエラーが返されませんでした")
	}
	// エラーの種類を確認 (net/http.Client.Timeout exceeded)
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "Timeout exceeded") { // Goのバージョン等でメッセージが異なる可能性
		t.Errorf("期待されるタイムアウトエラーメッセージではありません: %v", err)
	}
	t.Logf("期待通りタイムアウトエラーを検出: %v", err)
}

func TestPostToConfirmAPI_EmptyServerAddress(t *testing.T) {
	jsonData, _ := convertToJSON(testValidSheet)
	_, err := postToConfirmAPI(jsonData, "") // 空のアドレス
	if err == nil {
		t.Fatal("サーバーアドレスが空の場合にエラーが返されませんでした")
	}
	if !strings.Contains(err.Error(), "サーバーアドレスが空です") {
		t.Errorf("期待されるエラーメッセージではありません: %v", err)
	}
}

// --- handleAPIResponse のテスト ---

func TestHandleAPIResponse_Success(t *testing.T) {
	index := 0
	jsonResponse := []byte(`{
		"msg": "チェック完了、エラーあり",
		"sha256": "fedcba",
		"errors": [
			{"msg": "品番未登録", "details": "PN-XXX", "key": "Pid", "index": 0},
			{"msg": "数量不足", "key": "Quantity"}
		]
	}`)
	// "sheet" フィールドは含まれていない

	expectedResponse := PNResponse{ // 期待値
		Message: "チェック完了、エラーあり",
		SHA256:  "fedcba",
		Error: []ErrorRecord{
			{Message: "品番未登録", Details: "PN-XXX", Key: "Pid", Index: &index},
			{Message: "数量不足", Key: "Quantity"},
		},
		Sheet: Sheet{}, // JSONにsheetがないので、デコード後はゼロ値になるはず
	}

	actualResponse, err := handleAPIResponse(jsonResponse)
	if err != nil {
		t.Fatalf("予期せぬエラー: %v", err)
	}

	// まず全体を DeepEqual で比較試行
	if !reflect.DeepEqual(expectedResponse, actualResponse) {
		t.Errorf("PNResponseが期待値と異なります。")
		// --- 以下、詳細比較 ---
		if expectedResponse.Message != actualResponse.Message {
			t.Errorf("  Message: 期待=%q, 実際=%q", expectedResponse.Message, actualResponse.Message)
		}
		if expectedResponse.SHA256 != actualResponse.SHA256 {
			t.Errorf("  SHA256: 期待=%q, 実際=%q", expectedResponse.SHA256, actualResponse.SHA256)
		}
		// Error スライスの比較
		if len(expectedResponse.Error) != len(actualResponse.Error) {
			t.Errorf("  Error件数: 期待=%d, 実際=%d", len(expectedResponse.Error), len(actualResponse.Error))
		} else {
			for i := range expectedResponse.Error {
				// ErrorRecord 自体は比較可能なので DeepEqual
				if !reflect.DeepEqual(expectedResponse.Error[i], actualResponse.Error[i]) {
					t.Errorf("  Error[%d]: 期待=%+v, 実際=%+v", i, expectedResponse.Error[i], actualResponse.Error[i])
				}
			}
		}
		// Sheet の比較 (期待値も実際値もゼロ値のはず)
		if !reflect.DeepEqual(expectedResponse.Sheet, actualResponse.Sheet) {
			// 期待値は Sheet{} なので、実際値が Sheet{} でない場合のみエラーとする
			if !reflect.DeepEqual(Sheet{}, actualResponse.Sheet) {
				t.Errorf("  Sheet: 期待値はゼロ値ですが、実際値にはデータが含まれています: %+v", actualResponse.Sheet)
			}
		}
	}
}

func TestHandleAPIResponse_Success_NoErrorField(t *testing.T) {
	jsonResponse := []byte(`{"msg":"チェックOK","sha256":"abcde"}`) // errorsフィールドなし
	expectedResponse := PNResponse{
		Message: "チェックOK",
		SHA256:  "abcde",
		Error:   nil, // errorsがない場合はnilになるはず
	}

	actualResponse, err := handleAPIResponse(jsonResponse)
	if err != nil {
		t.Fatalf("予期せぬエラー: %v", err)
	}
	if !reflect.DeepEqual(expectedResponse, actualResponse) {
		t.Errorf("PNResponseが期待値と異なります。\n期待値: %+v\n実際値: %+v", expectedResponse, actualResponse)
	}
}

func TestHandleAPIResponse_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"msg": "チェックOK", sha256: "abc"`) // 不正なJSON
	_, err := handleAPIResponse(invalidJSON)
	if err == nil {
		t.Fatal("不正なJSONの場合にエラーが返されませんでした")
	}
	if !strings.Contains(err.Error(), "APIレスポンスJSONのデコードに失敗しました") {
		t.Errorf("期待されるエラーメッセージではありません: %v", err)
	}
	t.Logf("期待通りJSONデコードエラーを検出: %v", err)
}

func TestHandleAPIResponse_EmptyJSON(t *testing.T) {
	emptyJSON := []byte(`{}`)
	expectedResponse := PNResponse{} // 全てゼロ値/nilになるはず
	actualResponse, err := handleAPIResponse(emptyJSON)
	if err != nil {
		t.Fatalf("空のJSONで予期せぬエラー: %v", err)
	}
	if !reflect.DeepEqual(expectedResponse, actualResponse) {
		t.Errorf("空JSONに対するPNResponseが期待値と異なります。\n期待値: %+v\n実際値: %+v", expectedResponse, actualResponse)
	}
}
