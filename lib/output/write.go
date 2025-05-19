package output

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// templateFile : 出力するレポートのテンプレートファイルのパス
// embded するファイルパスと同じである必要がある
const templateFile = "report.tmpl"

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

type Report struct {
	Filename, Link, ErrorMessage string
	// []ErrorRecord  // TODO
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

// LogFatalError : エラーの内容をエラーログファイルに追記する
func LogFatalError(f string, msg string) error {
	// O_APPEND: ファイルの末尾に書き込む
	// O_CREATE: ファイルが存在しない場合は作成する
	// O_WRONLY: 書き込み専用で開く
	file, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	now := time.Now().Format("2006/01/02 15:04:05.000")
	msg = fmt.Sprintf("%s: %s\n", now, msg)
	_, err = file.WriteString(msg)
	return err
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
