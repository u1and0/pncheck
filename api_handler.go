package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// convertToJSON は Sheet 構造体を JSON バイトスライスに変換します。
func convertToJSON(sheet Sheet) ([]byte, error) {
	jsonData, err := json.Marshal(sheet)
	if err != nil {
		return nil, fmt.Errorf("Sheet構造体のJSON変換に失敗しました: %w", err)
	}
	return jsonData, nil
}

// postToConfirmAPI は指定されたJSONデータをAPIサーバーにPOSTし、
// レスポンスボディ、HTTPステータスコード、エラーを返します。
// ステータスコードが2xx以外でも、ボディがあれば読み込んで返します。
func postToConfirmAPI(jsonData []byte, serverAddress string) (body []byte, statusCode int, err error) {
	statusCode = -1 // 不明なステータスを表す初期値

	if serverAddress == "" {
		return nil, statusCode, errors.New("APIサーバーアドレスが空です")
	}
	apiURL := serverAddress + apiEndpointPath

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

// handleAPIResponse は APIレスポンスボディ (JSON) を PNResponse 構造体にデコードします。
func handleAPIResponse(responseBody []byte) (PNResponse, error) {
	var pnResponse PNResponse
	if err := json.Unmarshal(responseBody, &pnResponse); err != nil {
		return PNResponse{}, fmt.Errorf("APIレスポンスJSONのデコードに失敗しました: %w", err)
	}
	return pnResponse, nil
}
