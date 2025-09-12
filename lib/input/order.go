package input

import (
	"path/filepath"
	"strings"
	"time"
)

type OrderType string

const (
	購入  OrderType = "購入"
	外注  OrderType = "外注"
	出庫  OrderType = "出庫"
	未選択 OrderType = "未選択" // 不正な区分の場合
)

// parseOrderType ファイル名を引数に、"-"で区切った最後のブロックの値で発注区分を返す
// S: 出庫 OrderType = "出庫"
// K: 購入 OrderType = "購入"
// G: 外注 OrderType = "外注"
// それ以外: 未選択 OrderType = "未選択" // 不正な区分の場合
func parseOrderType(filePath string) OrderType {
	base := filepath.Base(filePath)
	// "-"で区切って4ブロック目の最初の文字
	blocks := strings.Split(base, "-")
	// fmt.Println("[DEBUG] parseOrderType() split filename: ", blocks)
	if len(blocks) < 4 {
		return "未選択"
	}
	lastBlock := blocks[3]
	// OrderTypeを決定
	switch {
	case strings.HasPrefix(lastBlock, "S"):
		return 出庫
	case strings.HasPrefix(lastBlock, "K"):
		return 購入
	case strings.HasPrefix(lastBlock, "G"):
		return 外注
	default:
		return 未選択
	}
}

func parseFilenameDate(filePath string) (time.Time, error) {
	const layout = "20060102"
	b := filepath.Base(filePath)
	ss := strings.Split(b, "-")
	return time.Parse(layout, ss[0])
}
