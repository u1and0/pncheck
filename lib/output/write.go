/*
output パッケージでは、
HTMLファイルへの出力や、ファイル拡張子の扱いを決定します。
*/
package output

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	// templateFile : 出力するレポートのテンプレートファイルのパス
	// embded するファイルパスと同じである必要がある
	templateFile = "report.tmpl"
)

//go:embed report.tmpl
var templateFS embed.FS

// StatusCode : HTTP status code 200～500番台
type StatusCode int

// Define a set of status codes using iota
const (
	successCode StatusCode = 200 + iota*100
	warningCode            // 300
	errorCode              // 400
	fatalCode              // 500
)

// Report : HTMLに表示するためのデータを纏めた構造体
// ファイル名やPNSearch表示用URLをまとめた構造体
type Report struct {
	Filename, Link string
	ErrorMessages  []string
	StatusCode
	// []ErrorRecord  // TODO 保存しておくと後で役立つかも？
	// Sheet // TODO 保存しておくと後で役立つかも？シートの修正とか。
}

// Reports : 各Report をステータスコードによって分類し、実行時間を格納しておく構造体
type Reports struct {
	// プログラムのビルド情報
	Version,
	ExecutionTime,
	BuildTime,
	ServerAddress string
	// エラーリポート
	SuccessItems,
	WarningItems,
	ErrorItems,
	FatalItems []Report
}

// Publish : report.tmplを基にReportsをHTMLファイルとして出力する
func (reports *Reports) Publish(outputPath string) error {
	tmpl, err := template.ParseFS(templateFS, templateFile)
	if err != nil {
		return err
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return tmpl.Execute(out, reports)
}

// ToJSON : Reports構造体をJSONとしてバイト列で返す
func (r *Reports) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// Classify : Reportに埋め込まれたHTTPステータスコードに基づいて分類
//
// @errors:
//
//	unknown status code %d: must 200 <= code < 600
func (reports *Reports) Classify(report Report) error {
	var c = report.StatusCode
	if c >= fatalCode {
		reports.FatalItems = append(reports.FatalItems, report)
	} else if c >= errorCode {
		reports.ErrorItems = append(reports.ErrorItems, report)
	} else if c >= warningCode {
		reports.WarningItems = append(reports.WarningItems, report)
	} else if c >= successCode {
		reports.SuccessItems = append(reports.SuccessItems, report)
	} else { // ステータス200未満、つまり初期値(0)のまま場合
		return fmt.Errorf("unknown status code %d: must 200 <= code < 600", c)
	}
	return nil
}

// ModifyFileExt
// filePathにはディレクトリが含まれており、
// ディレクトリとファイル名を分離して、
// ファイルの拡張子をセットし直す
func ModifyFileExt(filePath, newExt string) string {
	var (
		// ディレクトリとファイル名と拡張子を分離
		dir      = filepath.Dir(filePath)
		fileBase = WithoutFileExt(filePath)
	)
	return filepath.Join(dir, fileBase+newExt)
}

// WithoutFileExt : ディレクトリパスと拡張子を取り除く
func WithoutFileExt(filePath string) string {
	var (
		// ディレクトリとファイル名と拡張子を分離
		fileName = filepath.Base(filePath)
		ext      = filepath.Ext(filePath)
	)
	return strings.TrimSuffix(fileName, ext)

}
