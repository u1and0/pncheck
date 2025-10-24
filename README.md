指定されたExcelファイルをPNSearch APIでチェックします。

## 💡 Usage

### 💻 from command line

```sh
$ pncheck [オプション] <Excelファイルパス1> [Excelファイルパス2] ...
```

### ⚙️ Options:

- -V    レポートの詳細を表示します
- -VV APIの戻り値を表示します
- -VVV Excelシートへの入力を表示します
- -h,-help    ヘルプメッセージを表示します
- -v, -version    バージョン情報を表示します


### 📝 Example:

```sh
$ pncheck request1.xlsx request2.xlsx
```

### 📂 エクスプローラーから使う

![エクセルファイルをまとめてexe上にドラッグしてください。](doc/screen_shot_usage.png)

![HTMLファイルが出力されます。エラーやワーニングの具体的な内容はファイル名をクリックすると、ブレイクダウンされて表示されます。また、詳細ボタンを押すことで開かれるPNSearchを開くと、直接修正することができます。](doc/screen_shot_result.png)


## ☢️エラーの内容について

### PNSearchが検査する項目
PNSearchのヘルプを確認してください。

### pncheckが検査する項目
サーバー側で確認できないエラーはpncheck側で確認してFatalを発行します。
そのため、PNSearch のError, Warningを確認することができません。

- 行の順序にソートがかけられていること(納期順 -> 品番順)
- 要求票の版番号(バージョン)がPNSearchで作成されるものとと同一であること

#### Fatalが出た場合の確認項目
- Excelが読み込めない場合
- PNSearchと通信できない場合
- PNSearchからの応答に異常が含まれている場合


## 🏗️ Build

基本的に`make` を実行してください。
`make`で実行されるコマンドはMakefileのコメントと以下の解説を確認してください。

PNSearchのAPIを利用してる都合上、コマンドにサーバーアドレスを変数として注入してビルドします。
ビルドする際は環境に合った`SERVER_ADDRESS`変数を変更してビルドしてください。

```sh
$ make SERVER_ADDRESS='http://192.168.1.2:8080'
```

以降、`make`を使わずに個別にビルドしたい要望の場合について記述します。


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
