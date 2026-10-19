package main

import (
	"context"
	"cybertown/internal/game"
	"cybertown/internal/rouster"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
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

	err = game.InitWorld(ctx, db, config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("世界初始化完成，已有存档保持不变")

	fmt.Printf("配置加载成功：世界=%s，玩家=%d，NPC=%d\n",
		config.World.ID,
		len(config.Players),
		len(config.NPCs))

	fmt.Println("数据库表结构已创建：data/game.db")

	player, err := game.GetPlayer(ctx, db, "town-1", "player-1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"玩家：%s，场景：%s，位置：(%.2f, %.2f)，楼层：%d，版本：%d\n",
		player.Name,
		player.SceneID,
		player.Position.X,
		player.Position.Y,
		player.Floor,
		player.Revision,
	)

	r := gin.Default()
	rouster.SetUpRousters(r, db)

}
