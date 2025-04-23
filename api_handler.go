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

// postToConfirmAPI は指定されたJSONデータをAPIサーバーにPOSTし、レスポンスボディを返します。
// serverAddress は "http://host:port" の形式です。
func postToConfirmAPI(jsonData []byte, serverAddress string) ([]byte, error) {
	// APIの完全なURLを生成
	if serverAddress == "" {
		return nil, errors.New("APIサーバーアドレスが空です")
	}
	// 例: "http://localhost:8080/api/v1/requests/confirm"
	apiURL := serverAddress + apiEndpointPath

	// POSTリクエストを作成
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTPリクエストの作成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// 必要であれば他のヘッダー (認証トークンなど) も設定

	// HTTPクライアントを作成 (タイムアウト設定)
	client := &http.Client{Timeout: defaultTimeout}

	// リクエスト実行
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("APIへのリクエスト送信に失敗しました (%s): %w", apiURL, err)
	}
	defer resp.Body.Close()

	// ステータスコードをチェック (2xx以外はエラー)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// エラーレスポンスボディも読み取ってみる (エラー詳細が含まれる場合がある)
		bodyBytes, readErr := io.ReadAll(resp.Body)
		errorMsg := fmt.Sprintf("APIがエラーを返しました (ステータス: %d)", resp.StatusCode)
		if readErr == nil && len(bodyBytes) > 0 {
			errorMsg += fmt.Sprintf(" - レスポンス: %s", string(bodyBytes))
		}
		return nil, errors.New(errorMsg)
	}

	// レスポンスボディを読み込む
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("APIレスポンスボディの読み込みに失敗しました: %w", err)
	}

	return responseBody, nil
}

// handleAPIResponse は APIレスポンスボディ (JSON) を PNResponse 構造体にデコードします。
func handleAPIResponse(responseBody []byte) (PNResponse, error) {
	var pnResponse PNResponse
	if err := json.Unmarshal(responseBody, &pnResponse); err != nil {
		return PNResponse{}, fmt.Errorf("APIレスポンスJSONのデコードに失敗しました: %w", err)
	}
	return pnResponse, nil
}
