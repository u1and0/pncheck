/*
ビルド時に注入する変数 Makefile参照
*/

package input

var (
	// APIサーバーのアドレス http://localhost:8080 (ビルド時に注入)
	ServerAddress string
	// このCLIをビルドした日時 (ビルド時に注入)
	BuildTime string
)
