package game

import (
	"context"
	"database/sql"
	"fmt"
	"math"
)

func SavePlayerPosition(ctx context.Context, db *sql.DB, worldID string, playerID string, target Position, expectedRevision int64) (Player, error) {

	if worldID == "" || playerID == "" {
		return Player{}, fmt.Errorf("world id and player id are required")
	}

	if expectedRevision < 1 {
		return Player{}, fmt.Errorf("expected revision must be positive")
	}

	if math.IsNaN(target.X) || math.IsNaN(target.Y) || math.IsInf(target.X, 0) || math.IsInf(target.Y, 0) {
		return Player{}, fmt.Errorf("target position must be finite")
	}

	player, err := GetPlayer(ctx, db, worldID, playerID)
	if err != nil {

	}

	if player.Revision != expectedRevision {
		return Player{}, fmt.Errorf("player cannot move while status is %v", player.Status)
	}

	if player.Position == target {
		return player, nil
	}

	err = UpadtePlayerPosition(ctx, db, player.WorldID, playerID, target, player.Floor, expectedRevision)
	if err != nil {
		return Player{}, fmt.Errorf("move player: %w", err)
	}

	player.Position = target
	player.Revision++

	return player, nil
}
