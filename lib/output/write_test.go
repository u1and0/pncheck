package output

import (
	"bytes"
	"log"
	"os"
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
