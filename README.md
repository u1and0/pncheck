指定されたExcelファイルをPNSearch APIでチェックします。

## Usage

### from command line

```sh
$ pncheck [オプション] <Excelファイルパス1> [Excelファイルパス2] ...
```

### Options:
* -h, -help
    * ヘルプメッセージを表示します
* -v, -version
    * バージョン情報を表示します

### Example:

```sh
$ pncheck request1.xlsx request2.xlsx
```

### エクスプローラーから使う

![エクセルファイルをまとめてexe上にドラッグしてください。](doc/screen_shot_usage.png)

![エラーがあった場合にのみ、JSONファイルが出力されます。エラーの内容は生成されたJSONファイルを確認してください。](doc/screen_shot_result.png)


### Error JSON
エラーが書き込まれたJSONファイルの内容の一部

- msg: エラーの概要
- errors: エラーの詳細
    - message: エラーの詳細メッセージ
    - details: エラーの項目名
    - key: 列名
    - index: 行番号
- sha256: (使用しない)PNSearchで再表示するためのリンク
- sheet: (使用しない)読み込んだExcelの内容

```json
{
    "response": {
        "msg": "シートの確認でエラーが発生しました。",
        "errors": [
            {
                "message": "製番マスターに登録されていない情報です。製番名称、製番納期を確認してください。",
                "details": "準備材料費",
                "key": "製番名称"
            },
            {
                "message": "無効な品番です",
                "details": "製番 000079010741000 では品番 S_ZAIRYO を使えません。代わりに S_がつかない品番 を使ってください。",
                "key": "品番",
                "index": 4
            }
        ],
        "sha256": "a89fcb3ad46ae7cd235098a51bb047186861e0b426e243d6c9eca75ad0af8caa",
        "sheet": {
            "config": {
                "validatable": true,
                "sortable": true
            },
            "header": {
                "発注区分": "購入",
                "製番": "000079010741000",
```



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

#### icon

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


### Create Doc
README.mdをpandocでHTML形式に変換します。

```sh
$ pandoc README.md -o README.html
```
