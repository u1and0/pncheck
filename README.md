指定されたExcelファイルをPNSearch APIでチェックします。

## 💡 Usage

### 💻 from command line

```sh
$ pncheck [オプション] <Excelファイルパス1> [Excelファイルパス2] ...
```

### ⚙️ Options:
* -h, -help
    * ヘルプメッセージを表示します
* -v, -version
    * バージョン情報を表示します

### 📝 Example:

```sh
$ pncheck request1.xlsx request2.xlsx
```

### 📂 エクスプローラーから使う

![エクセルファイルをまとめてexe上にドラッグしてください。](doc/screen_shot_usage.png)

![HTMLファイルが出力されます。エラーの内容は生成されたリンク先へ飛んでから、PNSearchの[作成内容を確認する]ボタンを押してください。](doc/screen_shot_result.png)




## 🏗️ Build

`make` を実行してください。
`make`で実行されるコマンドはMakefileのコメントと以下の解説を確認してください。

PNSearchのAPIを利用してる都合上、コマンドにサーバーアドレスを変数として注入してビルドします。
ビルドする際は環境に合った`SERVER_ADDRESS`変数を変更してビルドしてください。

```sh
$ make SERVER_ADDRESS='http://192.168.1.2:8080'
```

### 🐧 for Linux

`go build`を実行します。 ビルド時にPNSearchサーバーのURLを決定します。

```sh
$ SERVER_ADDRESS='http://localhost:8080'
$ go build -ldflags="-X pncheck/lib/input.ServerAddress=${SERVER_ADDRESS}"
```


### 🪟 for Windows

環境変数 GOOSと GOARCHを設定してからビルドします。
あるいは以下のように、on the flyで環境変数を設定してからビルドします。

```sh
$ GOOS=windows GOARCH=amd64 go build -ldflags="-X pncheck/lib/input.ServerAddress=${SERVER_ADDRESS}"

```

#### 🎨 icon

go-winresというツールでアイコンを埋め込みます。

```sh
$ go install github.com/tc-hib/go-winres@latest
$ go-winres init
```

winresディレクトリにサンプルファイルが配置されるので、 icon.png, icon16.pngの差し替えます。

```sh
$ go-winres make
```

.sysoファイルが作成されます。この状態で通常通り`go build .`(このプロジェクトの場合は`make exe`)をするとアイコン付きのexeが生成されます。


### 📄 Create Doc
README.mdをpandocでHTML形式に変換します。

```sh
$ pandoc README.md -o README.html
```
