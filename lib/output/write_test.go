package output

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"  // assertを使うと簡潔に書けます
	"github.com/stretchr/testify/require" // requireを使うとテスト失敗時に即座に停止します
)

// captureLog captures log output written by log.Printf and returns it as a string.
func captureLog(t *testing.T, f func()) string {
	// 標準ロガーの出力をキャプチャするためのバッファ
	var buf bytes.Buffer
	// 元の出力先を保存
	originalOutput := log.Writer()
	// ロガーの出力先をバッファに設定
	log.SetOutput(&buf)
	// テスト終了時に元の出力先に戻す
	defer log.SetOutput(originalOutput)

	// テスト対象の関数を実行
	f()

	// バッファの内容を文字列として返す
	return buf.String()
}

// setupTestDir creates a temporary directory for testing and returns its path.
// It also schedules the deletion of the directory after the test finishes.
func setupTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "output_test")
	require.NoError(t, err, "Failed to create temporary directory")

	t.Cleanup(func() {
		// ディレクトリとその内容を削除
		err := os.RemoveAll(dir)
		assert.NoError(t, err, "Failed to remove temporary directory %s", dir)
	})

	return dir
}

func TestWriteErrorToJSON(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		testDir := setupTestDir(t)
		jsonPath := filepath.Join(testDir, "error.json")
		body := []byte(`{"error": "something went wrong"}`)

		err := WriteErrorToJSON(jsonPath, body)
		assert.NoError(t, err, "WriteErrorToJSON should not return an error on success")

		// ファイルが存在することを確認
		_, err = os.Stat(jsonPath)
		assert.NoError(t, err, "JSON file should exist after writing")

		// ファイル内容を確認
		content, err := os.ReadFile(jsonPath)
		assert.NoError(t, err, "Failed to read written file")
		assert.Equal(t, body, content, "File content should match the provided body")
	})

	t.Run("Failure to Create Directory", func(t *testing.T) {
		testDir := setupTestDir(t)
		// 存在しないディレクトリを指定
		jsonPath := filepath.Join(testDir, "nonexistent_dir", "error.json")
		body := []byte(`{"error": "test"}`)

		err := WriteErrorToJSON(jsonPath, body)
		assert.Error(t, err, "WriteErrorToJSON should return an error when directory creation fails")
		assert.Contains(t, err.Error(), "エラーファイル", "Error message should be in Japanese")
		// Windowsでは"指定されたパスが見つかりません。", Linux/macOSでは"no such file or directory"などになる
		// OS依存のため詳細なエラーメッセージは確認しないか、抽象的に確認する
		// assert.Contains(t, err.Error(), "no such file or directory") // 例 (OS依存)

		// ファイルが作成されていないことを確認 (Statはエラーになる)
		_, err = os.Stat(jsonPath)
		assert.True(t, os.IsNotExist(err), "JSON file should not be created")
	})
}

func TestLogFatalError(t *testing.T) {
	t.Run("Success_NewFileAndAppend", func(t *testing.T) {
		testDir := setupTestDir(t)
		logPath := filepath.Join(testDir, "app.log")
		msg1 := "First error message"
		msg2 := "Second error message"

		// 最初のメッセージを書き込み (ファイルが新規作成されるはず)
		err := LogFatalError(logPath, msg1)
		assert.NoError(t, err, "LogFatalError should not return an error on first write")

		// ファイルが存在することを確認
		_, err = os.Stat(logPath)
		assert.NoError(t, err, "Log file should exist after first write")

		// ファイル内容を確認 (最初のメッセージ)
		contentBytes, err := os.ReadFile(logPath)
		assert.NoError(t, err, "Failed to read log file after first write")
		contentLines := strings.Split(string(contentBytes), "\n")
		// 最後の要素は空行または不完全な行の可能性があるため、最後の要素を除外して確認
		if contentLines[len(contentLines)-1] == "" {
			contentLines = contentLines[:len(contentLines)-1]
		}
		require.Len(t, contentLines, 1, "Log file should have 1 line after first write")
		// タイムスタンプ形式とメッセージが含まれていることを確認
		assert.Contains(t, contentLines[0], ": "+msg1, "First line should contain the first message")
		// タイムスタンプ部分の形式を簡易的に確認 (YYYY/MM/DD HH:MM:SS.sss)
		parts := strings.SplitN(contentLines[0], ": ", 2)
		require.Len(t, parts, 2, "Log line format should be 'timestamp: message'")
		_, err = time.Parse("2006/01/02 15:04:05.000", parts[0])
		assert.NoError(t, err, "Timestamp format should be 'YYYY/MM/DD HH:MM:SS.sss'")

		// 2番目のメッセージを書き込み (ファイルに追記されるはず)
		err = LogFatalError(logPath, msg2)
		assert.NoError(t, err, "LogFatalError should not return an error on second write")

		// ファイル内容を確認 (2つのメッセージ)
		contentBytes, err = os.ReadFile(logPath)
		assert.NoError(t, err, "Failed to read log file after second write")
		contentLines = strings.Split(string(contentBytes), "\n")
		// 最後の要素は空行または不完全な行の可能性があるため、最後の要素を除外して確認
		if contentLines[len(contentLines)-1] == "" {
			contentLines = contentLines[:len(contentLines)-1]
		}
		require.Len(t, contentLines, 2, "Log file should have 2 lines after second write")

		assert.Contains(t, contentLines[0], ": "+msg1, "First line should still contain the first message")
		assert.Contains(t, contentLines[1], ": "+msg2, "Second line should contain the second message")

		// 各行のタイムスタンプが昇順になっているか (大まかに)
		ts1Str := strings.SplitN(contentLines[0], ": ", 2)[0]
		ts2Str := strings.SplitN(contentLines[1], ": ", 2)[0]
		ts1, _ := time.Parse("2006/01/02 15:04:05.000", ts1Str)
		ts2, _ := time.Parse("2006/01/02 15:04:05.000", ts2Str)
		assert.True(t, ts2.After(ts1) || ts2.Equal(ts1), "Second timestamp should be after or equal to first timestamp")
	})

	t.Run("Failure to Open File", func(t *testing.T) {
		testDir := setupTestDir(t)
		// 存在しないディレクトリを指定
		logPath := filepath.Join(testDir, "nonexistent_dir", "app.log")
		msg := "Some error"

		err := LogFatalError(logPath, msg)
		assert.Error(t, err, "LogFatalError should return an error when file opening fails")
		// OS依存のエラーメッセージは確認しないか、抽象的に確認する
		// assert.Contains(t, err.Error(), "no such file or directory") // 例 (OS依存)

		// ファイルが作成されていないことを確認 (Statはエラーになる)
		_, err = os.Stat(logPath)
		assert.True(t, os.IsNotExist(err), "Log file should not be created")
	})

	// Note: Writing failure simulation is hard without mocking os package.
	// Skipping for a standard test.
	t.Run("Write Failure", func(t *testing.T) {
		// Simulate a write failure, e.g., disk full, permissions.
		// This is hard to reliably simulate in a standard unit test.
		// One approach could be to write to a pipe closed at the other end,
		// but that requires lower-level OS interaction.

		t.Log("Skipping explicit test for write failure in LogFatalError due to complexity")
	})

}
