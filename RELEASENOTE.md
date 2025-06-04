# Release note v1.3.2

## バグ修正
- 複数の日付型の解釈を適用した
```
dateLayoutSub  = []string{"01-02-06", "2006/1/2", "1/2/2006"} // PNSearch規格外の日付文字列
```
