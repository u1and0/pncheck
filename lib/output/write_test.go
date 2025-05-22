package output

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"testing"

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
