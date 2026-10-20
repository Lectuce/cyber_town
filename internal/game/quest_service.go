package game

import (
	"context"
	"database/sql"
	"fmt"
)

func AcceptQuest(ctx context.Context, db *sql.DB, worldID string, playerID string, questID string) (PlayerQuest, error) {
	if worldID == "" || playerID == "" || questID == "" {
		return PlayerQuest{}, fmt.Errorf(
			"world id, player id and quest id are required",
		)
	}

	definition, err := GetQuestDefinition(questID)
	if err != nil {
		return PlayerQuest{}, err
	}

	player, err := GetPlayer(ctx, db, worldID, playerID)
	if err != nil {
		return PlayerQuest{}, err
	}

	if player.Status != "active" {
		return PlayerQuest{}, fmt.Errorf("player cannot accept quests while status is %q", player.Status)
	}

	// 确认这个世界中确实存在任务发布者。
	_, err = GetNPC(ctx, db, worldID, definition.GiverNPCID)
	if err != nil {
		return PlayerQuest{}, fmt.Errorf("load quest giver: %w", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO player_quests (
			world_id, player_id, quest_id, status, revision
		)
		VALUES (?, ?, ?, 'active', 1)
		ON CONFLICT(world_id, player_id, quest_id) DO NOTHING
	`,
		worldID,
		playerID,
		questID,
	)
	if err != nil {
		return PlayerQuest{}, fmt.Errorf("accept quest: %w", err)
	}

	return GetPlayerQuest(
		ctx,
		db,
		worldID,
		playerID,
		questID,
	)
}
