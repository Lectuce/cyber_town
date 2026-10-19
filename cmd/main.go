package main

import (
	"context"
	"cybertown/internal/game"
	"fmt"
	"log"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := game.LoadTownConfig("config/town/town.json")
	if err != nil {
		log.Fatal(err)
	}

	db, err := game.OpenDatabase(ctx, "data/game.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Printf("配置加载成功：世界=%s，玩家=%d，NPC=%d\n",
		config.World.ID,
		len(config.Players),
		len(config.NPCs))

	fmt.Println("数据库表结构已创建：data/game.db")

}
