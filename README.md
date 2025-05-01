## Build

$ go build -ldflags="-X pncheck/lib/input.serverAddress=http://localhost:8080"

ビルド時にPNSearchサーバーのURLを決定します。


### for Windows

環境変数 GOOSと GOARCHを設定してからビルドします。
あるいは以下のように、on the flyで環境変数を設定してからビルドします。

```
$ GOOS=windows GOARCH=amd64 go build -ldflags="-X pncheck/lib/input.serverAddress=http://localhost:8080"

```
