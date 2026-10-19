package game

import (
	"context"
	"database/sql"
	"fmt"
)

func GetNPC(
	ctx context.Context,
	db *sql.DB,
	worldID string,
	npcID string,
) (NPC, error) {
	var npc NPC

	err := db.QueryRowContext(ctx,
		`
		SELECT
			world_id, id, name, role, persona, background,
			scene_id, current_activity,
			position_x, position_y, floor, revision
		FROM npcs
		WHERE world_id = ? AND id = ?
	`, worldID, npcID).Scan(
		&npc.WorldID,
		&npc.ID,
		&npc.Name,
		&npc.Role,
		&npc.Persona,
		&npc.Background,
		&npc.SceneID,
		&npc.CurrentActivity,
		&npc.Position.X,
		&npc.Position.Y,
		&npc.Floor,
		&npc.Revision,
	)

	if err != nil {
		return NPC{}, fmt.Errorf(
			"get npc %q in world %q: %w",
			npcID,
			worldID,
			err,
		)
	}

	return npc, nil
}
