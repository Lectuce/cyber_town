package game

import (
	"context"
	"database/sql"
	"fmt"
	"math"
)

func SavePlayerPosition(ctx context.Context, db *sql.DB, worldID string, playerID string, target Position, floor int, expectedRevision int64) (Player, error) {

	if worldID == "" || playerID == "" {
		return Player{}, fmt.Errorf("world id and player id are required")
	}

	if expectedRevision < 1 {
		return Player{}, fmt.Errorf("expected revision must be positive")
	}

	if floor < 0 || floor > 3 {
		return Player{}, fmt.Errorf("invalid floor: %d", floor)
	}

	if math.IsNaN(target.X) || math.IsNaN(target.Y) || math.IsInf(target.X, 0) || math.IsInf(target.Y, 0) {
		return Player{}, fmt.Errorf("target position must be finite")
	}

	player, err := GetPlayer(ctx, db, worldID, playerID)
	if err != nil {
		return Player{}, err
	}

	if player.Revision != expectedRevision {
		return Player{}, fmt.Errorf("player cannot move while status is %v", player.Status)
	}

	// 坐标和楼层都没有变化，就不用更新存档。
	if player.Position == target && player.Floor == floor {
		return player, nil
	}

	err = UpadtePlayerPosition(ctx, db, player.WorldID, playerID, target, player.Floor, expectedRevision)
	if err != nil {
		return Player{}, fmt.Errorf("move player: %w", err)
	}

	player.Position = target
	player.Floor = floor
	player.Revision++

	return player, nil
}
