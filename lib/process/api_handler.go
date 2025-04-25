package process

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	pnio "pncheck/lib/io"
)

type (
	ErrorRecord struct {
		Message string `json:"msg"`               // 例: "品番が見つかりません"
		Details string `json:"details,omitempty"` // 例: "PN-XXX"
		Key     string `json:"key,omitempty"`     // 例: "品番" or "Pid"
		Index   *int   `json:"index,omitempty"`   // エラーが発生した行番号 (0-based or 1-based? API仕様による)
	}

	PNResponse struct {
		Message string        `json:"msg"`              // ログの概要 (例: "チェック完了", "エラーあり")
		Error   []ErrorRecord `json:"errors,omitempty"` // エラーがあればErrorRecordを追記
		SHA256  string        `json:"sha256,omitempty"` // Sheet構造体から計算したsha256ハッシュ
		Sheet   pnio.Sheet    `json:"sheet,omitempty"`  // 検証対象のSheet構造体 (オプション)
	}
)

// APIのエンドポイントパス (固定値とする)
const apiEndpointPath = "/api/v1/requests/confirm"

var defaultTimeout = 30 * time.Second // API通信のデフォルトタイムアウト

// postToConfirmAPI は指定されたJSONデータをAPIサーバーにPOSTし、
// レスポンスボディ、HTTPステータスコード、エラーを返します。
// ステータスコードが2xx以外でも、ボディがあれば読み込んで返します。
func PostToConfirmAPI(sheet pnio.Sheet, serverAddress string) (body []byte, statusCode int, err error) {

	if serverAddress == "" {
		return nil, statusCode, errors.New("APIサーバーアドレスが空です")
	}
	apiURL := serverAddress + apiEndpointPath

	jsonData, err := json.Marshal(sheet)
	if err != nil {
		return nil, statusCode, fmt.Errorf("Sheet構造体のJSON変換に失敗しました: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, statusCode, fmt.Errorf("HTTPリクエストの作成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: defaultTimeout}
	resp, err := client.Do(req)
	if err != nil {
		// 接続エラーなど、レスポンス自体が得られなかった場合
		return nil, statusCode, fmt.Errorf("APIへのリクエスト送信に失敗しました (%s): %w", apiURL, err)
	}
	defer resp.Body.Close()

	// レスポンスが得られた場合はステータスコードを記録
	statusCode = resp.StatusCode

	// ボディを読み込む (ステータスコードに関わらず試みる)
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		// ボディ読み込み失敗は致命的エラー
		return nil, statusCode, fmt.Errorf("APIレスポンスボディの読み込みに失敗しました (ステータス: %d): %w", statusCode, readErr)
	}
	return body, statusCode, nil
}

// HandleAPIResponse は APIレスポンスボディ (JSON) を PNResponse 構造体にデコードします。
func HandleAPIResponse(responseBody []byte) (PNResponse, error) {
	var pnResponse PNResponse
	if err := json.Unmarshal(responseBody, &pnResponse); err != nil {
		return PNResponse{}, fmt.Errorf("APIレスポンスJSONのデコードに失敗しました: %w", err)
	}
	return pnResponse, nil
}
