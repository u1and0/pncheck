## Build

`make` を実行してください。
`make`で実行されるコマンドはMakefileのコメントと以下の解説を確認してください。

### for Linux

`go build`を実行します。 ビルド時にPNSearchサーバーのURLを決定します。

```sh
$ SERVER_ADDRESS='http://localhost:8080'
$ go build -ldflags="-X pncheck/lib/input.serverAddress=${SERVER_ADDRESS}"
```


### for Windows

環境変数 GOOSと GOARCHを設定してからビルドします。
あるいは以下のように、on the flyで環境変数を設定してからビルドします。

```sh
$ GOOS=windows GOARCH=amd64 go build -ldflags="-X pncheck/lib/input.serverAddress=${SERVER_ADDRESS}"

```


### Create Doc
README.mdをpandocでHTML形式に変換します。

```
pandoc README.md -o README.html
```
