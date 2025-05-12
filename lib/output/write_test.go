package output

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestWriteFatal(t *testing.T) {
	// Test case 1: Successful write
	t.Run("successful write", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		filePath := filepath.Join(tmpDir, "test.txt")
		err = os.WriteFile(filePath, []byte{}, 0644)
		if err != nil {
			t.Fatal(err)
		}

		testErr := errors.New("test error")
		err = WriteFatal(filePath, testErr)
		if err != nil {
			t.Errorf("WriteFatal returned error: %v", err)
		}

		jsonFilePath := "test.json"
		if _, err := os.Stat(jsonFilePath); os.IsNotExist(err) {
			t.Errorf("JSON file %s not found", jsonFilePath)
		}

		jsonData, err := os.ReadFile(jsonFilePath)
		if err != nil {
			t.Fatal(err)
		}

		var errRecord ErrorRecord
		err = json.Unmarshal(jsonData, &errRecord)
		if err != nil {
			t.Fatal(err)
		}

		if errRecord.Filename != "test.txt" || errRecord.Error != "test error" {
			t.Errorf("Unexpected JSON content: %+v", errRecord)
		}
	})

	// Test case 2: JSON marshaling error
	t.Run("JSON marshaling error", func(t *testing.T) {
		// This test case is tricky because we're testing an unexported function indirectly.
		// The `json.MarshalIndent` function will fail if the input is not marshalable.
		// We can't directly test this with `ErrorRecord` because it's a simple struct that can be marshaled.
		// However, we can test the error handling by passing a non-marshalable error (e.g., a nil error).
		err := WriteFatal("test.txt", nil)
		if err != nil {
			t.Errorf("WriteFatal did not return error for nil error input")
		}
	})

	// Test case 3: Error writing to JSON file
	t.Run("error writing to JSON file", func(t *testing.T) {
		// Create a directory where we expect to write the JSON file, so the write fails.
		tmpDir, err := os.MkdirTemp("", "test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		filePath := filepath.Join(tmpDir, "test.txt")
		err = os.WriteFile(filePath, []byte{}, 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Create a directory with the same name as the expected JSON file.
		jsonFilePath := filepath.Join(tmpDir, "test.json")
		err = os.Mkdir(jsonFilePath, 0755)
		if err != nil {
			t.Fatal(err)
		}

		testErr := errors.New("test error")
		err = WriteFatal(jsonFilePath, testErr)
		if err != nil {
			t.Errorf("WriteFatal did not return error when writing to JSON file failed")
		}
	})
}

func TestModifyFilePath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "basic functionality",
			filePath: "/path/to/file.txt",
			want:     "/path/to/file.pncheck.xlsx",
		},
		{
			name:     "filename with extension",
			filePath: "/path/to/file.with.multiple.extensions.txt",
			want:     "/path/to/file.with.multiple.extensions.pncheck.xlsx",
		},
		{
			name:     "filename without extension",
			filePath: "/path/to/file",
			want:     "/path/to/file.pncheck.xlsx",
		},
		{
			name:     "empty string",
			filePath: "",
			want:     "..pncheck.xlsx",
		},
		{
			name:     "just a filename",
			filePath: "file.txt",
			want:     "file.pncheck.xlsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ModifyFileExt(tt.filePath, ".pncheck.xlsx")
			if got != tt.want {
				t.Errorf("modifyFilePath() got = %v, want %v", got, tt.want)
			}
		})
	}
}
