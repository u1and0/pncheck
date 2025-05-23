package output

import (
	"embed"
	"fmt"
	"log"
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

// WriteErrorToJSON writes error response to a JSON file
func WriteErrorToJSON(jsonPath string, body []byte) error {
	f, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("エラーファイル '%s' の作成に失敗しました: %w", jsonPath, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// closeエラーをログに記録するか、既存のエラーに追加する
			// 通常、書き込みエラーの方が重要なので、closeエラーは単にログに記録する
			log.Printf("エラーファイル '%s' のクローズに失敗しました: %v", jsonPath, err)
		}
	}()

	if _, err := f.Write(body); err != nil {
		return fmt.Errorf("エラーファイル '%s' へのJSONデータ書き込みに失敗しました: %w", jsonPath, err)
	}

	return nil
}

// Define a custom type Status as an alias for int
type Status int

// Define a set of status codes using iota
const (
	successCode Status = 200 + iota*100
	warningCode        // 300
	errorCode          // 400
	fatalCode          // 500
)

type Report struct {
	Filename, Link, ErrorMessage string
	StatusCode                   Status
	// []ErrorRecord  // TODO 保存しておくと後で役立つかも？
	// Sheet // TODO 保存しておくと後で役立つかも？シートの修正とか。
}

type Reports struct {
	ExecutionTime                                      string
	SuccessItems, WarningItems, ErrorItems, FatalItems []Report
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

// Classify : Reportに埋め込まれたHTTPステータスコードに基づいて分類
func (reports *Reports) Classify(report Report) {
	if report.StatusCode >= errorCode && report.StatusCode < fatalCode {
		reports.ErrorItems = append(reports.ErrorItems, report)
	} else if report.StatusCode >= warningCode {
		reports.WarningItems = append(reports.WarningItems, report)
	} else if report.StatusCode >= successCode {
		reports.SuccessItems = append(reports.SuccessItems, report)
	} else { // report.StatusCode >= fatalCode  || reports.StatusCode < successCode{
		reports.FatalItems = append(reports.FatalItems, report)
	}
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
