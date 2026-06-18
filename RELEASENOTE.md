# Release note v1.6.15: トグル手動開閉と2回目POSTの偽Successの非表示

## 修正
- HTML template accordion 手動開閉
    - すべてのグループ(Fatal, Error, Warning, Success) は他のグループと干渉せず、すべて自動でトグル開閉する挙動に変更。
- Error出た後のWarningチェックで、Successが出てしまう問題を解消。
    - 2回目POSTのSuccess は偽物なので、reportに対して送信しない挙動に変更した。
