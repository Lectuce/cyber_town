package game

import "fmt"

// 物品本身的固定信息。
type ItemDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// 玩家背包中一种物品的展示信息。
type InventoryItem struct {
	ItemID   string `json:"item_id"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

func GetItemDefinition(itemID string) (ItemDefinition, error) {
	switch itemID {
	case "shipment_receipt":
		return ItemDefinition{
			ID:          "shipment_receipt",
			Name:        "商队货物凭证",
			Description: "在失踪货车现场取得，可交给商人核对货物。",
		}, nil

	case "coin":
		return ItemDefinition{
			ID:          "coin",
			Name:        "金币",
			Description: "完成委托获得的报酬。",
		}, nil

	default:
		return ItemDefinition{}, fmt.Errorf(
			"unknown item: %q",
			itemID,
		)
	}
}
