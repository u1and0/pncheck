package lib

import (
	"flag"
	"os"
	"reflect"
	"testing"
)

func TestParseArguments(t *testing.T) {
	oldArgs := os.Args
	// flag.CommandLine の元の状態を保存 (テスト全体で1回)
	originalCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		// テスト終了時に元の CommandLine に戻す
		flag.CommandLine = originalCommandLine
	}()

	tests := []struct {
		name      string
		args      []string
		wantPaths []string
		wantErr   bool
	}{
		{
			name:      "正常系 - 1ファイル",
			args:      []string{"testapp", "file1.xlsx"},
			wantPaths: []string{"file1.xlsx"},
			wantErr:   false,
		},
		{
			name:      "正常系 - 複数ファイル",
			args:      []string{"testapp", "file1.xlsx", "path/to/file2.xlsx", "file3.xlsx"},
			wantPaths: []string{"file1.xlsx", "path/to/file2.xlsx", "file3.xlsx"},
			wantErr:   false,
		},
		{
			name:      "異常系 - 引数なし",
			args:      []string{"testapp"},
			wantPaths: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			// 新しい FlagSet を作成し、テスト期間中だけ差し替える
			// flag.ExitOnError を使うとテストが中断してしまう可能性があるため、
			// flag.ContinueOnError を使うか、エラーをハンドリングする
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError) // または flag.PanicOnError

			gotPaths, err := ParseArguments("v0.1.0") // ここで flag.Parse() が呼ばれる

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArguments() error = %v, wantErr %v", err, tt.wantErr)
				// エラーが期待通りでない場合、flagパッケージからのエラーメッセージも確認すると良い
				// if err != nil { t.Logf("  flag error: %v", err) }
				return
			}
			// wantErr が true の場合、エラーの種類を検証しても良い
			// if tt.wantErr && err == nil { ... }

			if !reflect.DeepEqual(gotPaths, tt.wantPaths) {
				t.Errorf("ParseArguments() gotPaths = %v, want %v", gotPaths, tt.wantPaths)
			}
		})
	}
}

// 注意: -h や --help のテストは、os.Exit を呼び出すため単純にはテストできません。
// os.Exit をモック化する、またはコマンドの出力をキャプチャするような
// より高度なテスト手法が必要になります。
