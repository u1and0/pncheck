package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pncheck/lib/input"
	"pncheck/lib/output"
)

const (
	fatalLog = "fatal_report_log.json" // エラーレポートのデフォルトファイル名
)

// サーバーからのエラー出力のうち、このコマンドで使用するもの
type ErrorOutput struct {
	Filename string               `json:"filename"`
	Msg      string               `json:"msg"`
	Errors   []output.ErrorRecord `json:"errors,omitempty"`
}

// WriteErrorFile は集約されたエラー結果を指定されたファイルにJSON形式で出力します。
// エラーがない場合はファイルを作成しません。
// 出力形式: エラーがあった FileProcessResult のスライスをインデント付きJSONで出力
func (eo *ErrorOutput) WriteErrorFile(path string) error {
	// ファイルを開く (なければ作成、あれば上書き)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' の作成に失敗しました: %w", path, err)
	}
	defer file.Close()

	b, err := json.MarshalIndent(&eo, "", "  ")
	_, err = file.Write(b)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' へのJSONデータ書き込みに失敗しました: %w", path, err)
	}
	return nil
}

// ProcessExcelFile は1つのExcelファイルを処理し、その結果を FileProcessResult として返します。
// 内部でファイルの読み込み、JSON変換、API呼び出し、レスポンス処理を行います。
func ProcessExcelFile(filePath string) error {
	// Excel読み込み
	sheet, err := input.ReadExcelToSheet(filePath)
	if err != nil {
		return fmt.Errorf("Excel読み込みエラー: %w", err)
	}

	// API呼び出し
	body, code, err := sheet.Post()
	if err != nil {
		return fmt.Errorf("API通信エラー: %w", err)
	}
	// APIレスポンス解析とエラー出力 (ボディがあれば実行)
	if body == nil || len(body) < 1 {
		return fmt.Errorf("APIレスポンス解析エラー (ステータス: %d): %w", code, err)
	}

	fmt.Printf("Response %d: %s", code, body)

	// codeに対する処理を分岐
	// 200で成功なら何もしない
	// 300-400番台ステータスコードはファイル名.jsonにエラーの内容を書き込む
	// 500番台ステータスコードはPOSTに失敗しているので、faital_report.log にエラーを書き込み
	switch {
	// コンソールに成功メッセージを書くだけ
	case code < 300:
		fmt.Println("Success:", filePath)
		// _, fileName := path.Split(filePath)
		// err = os.Rename(filePath, path.Join("success", fileName))
		// if err != nil {
		// 	return fmt.Errorf("ファイル移動エラー: ファイルを移動できませんでした。", code, err)
		// }
		// xslxを移動してJSONファイルを書き込み
		// jsonPath := filepath.Join("success", jsonFile)
		// apiResponse.WriteJSON(jsonPath)

	// 300,400番台はPNResponseをJSONに書き込む
	case code < 500:
		jsonPath := filepath.Base(filePath) + ".json"
		f, err := os.Create(jsonPath)
		if err != nil {
			return fmt.Errorf("エラーファイル '%s' の作成に失敗しました: %w", jsonPath, err)
		}
		defer f.Close()

		// Parse JSON
		// apiResponse, err := output.HandleAPIResponse(body)
		// if err != nil {
		// 	return err
		// }
		// log.Printf("pnsearch response: %#v\n", apiResponse)
		//
		// b, err := json.MarshalIndent(&apiResponse, "", "  ")
		_, err = f.Write(body)
		if err != nil {
			return fmt.Errorf("エラーファイル '%s' へのJSONデータ書き込みに失敗しました: %w", jsonPath, err)
		}

	default:
		// fatal_report.logを開いて、エラーの内容を追記する。
		// すでにfatal_report.logが存在しても初期化しないで内容を追記する
		// ファイルが存在する場合、内容を読み取る
		var data []byte
		if _, err := os.Stat(fatalLog); err == nil {
			data, err = os.ReadFile(fatalLog)
			if err != nil {
				return err
			}
		}

		// エラーの内容を追記
		now := time.Now().Format("2006/01/02 15:04:05")
		msg := now + ": PNSearch /confirm API への通信に失敗しました。"
		if data != nil {
			data = append(data, []byte(msg)...)
		} else {
			data = []byte(msg)
		}

		// ファイルに書き込み
		err = os.WriteFile(fatalLog, data, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// エラーログを書き込む関数
func writeErrorLog(filename string, err error) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("%v\n", err))
	return err
}

func moveFile(src, dst string) error {
	// dstのディレクトリが存在しない場合は作成する
	dstDir := filepath.Dir(dst)
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		err = os.Mkdir(dstDir, 0755)
		if err != nil {
			return err
		}
	}

	// ファイルを移動させる
	err := os.Rename(src, dst)
	return err
}
