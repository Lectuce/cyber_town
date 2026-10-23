package game

import (
	"context"
	"database/sql"
	"fmt"
)

type NPCQuestStatus struct {
	QuestID                 string `json:"quest_id"`
	Title                   string `json:"title"`
	Status                  string `json:"status"`
	HasRequiredEvidence     bool   `json:"has_required_evidence"`
	RequiredItemID          string `json:"required_item_id"`
	RequiredItemCount       int64  `json:"required_item_count"`
	OwnedItemCount          int64  `json:"owned_item_count"`
	SubmissionConditionsMet bool   `json:"submission_conditions_met"`
	RewardItemID            string `json:"reward_item_id"`
	RewardItemCount         int64  `json:"reward_item_count"`
}

func GetNPCQuestStatus(
	ctx context.Context,
	db *sql.DB,
	worldID string,
	playerID string,
	npcID string,
) (NPCQuestStatus, error) {

	if worldID == "" || playerID == "" || npcID == "" {
		return NPCQuestStatus{}, fmt.Errorf(
			"world id, player id and NPC id are required",
		)
	}

	// 货车调查任务。
	definition, err := GetQuestDefinition("missing_iron_cart")
	if err != nil {
		return NPCQuestStatus{}, err
	}

	// 只允许任务发布者查询这份提交进度。
	if npcID != definition.GiverNPCID {
		return NPCQuestStatus{}, fmt.Errorf(
			"当前角色无权查询这项委托的提交进度",
		)
	}

	if _, err := GetNPC(ctx, db, worldID, npcID); err != nil {
		return NPCQuestStatus{}, fmt.Errorf("load NPC: %w", err)
	}

	result := NPCQuestStatus{
		QuestID:           definition.ID,
		Title:             definition.Title,
		RequiredItemID:    definition.RequiredItemID,
		RequiredItemCount: definition.RequiredItemCount,
		RewardItemID:      definition.RewardItemID,
		RewardItemCount:   definition.RewardItemCount,
	}

	var playerStatus string

	// 一条查询读取相关状态，避免分别查询时读到不同时间的结果。
	err = db.QueryRowContext(ctx, `
		SELECT
			p.status,
			COALESCE(q.status, 'not_started'),
			EXISTS (
				SELECT 1
				FROM player_evidence e
				WHERE e.world_id = p.world_id
					AND e.player_id = p.id
					AND e.evidence_id = ?
			),
			COALESCE((
				SELECT i.quantity
				FROM player_inventory i
				WHERE i.world_id = p.world_id
					AND i.player_id = p.id
					AND i.item_id = ?
			), 0)
		FROM players p
		LEFT JOIN player_quests q
			ON q.world_id = p.world_id
			AND q.player_id = p.id
			AND q.quest_id = ?
		WHERE p.world_id = ? AND p.id = ?
	`,
		definition.RequiredEvidenceID,
		definition.RequiredItemID,
		definition.ID,
		worldID,
		playerID,
	).Scan(
		&playerStatus,
		&result.Status,
		&result.HasRequiredEvidence,
		&result.OwnedItemCount,
	)
	if err != nil {
		return NPCQuestStatus{}, fmt.Errorf(
			"query NPC quest status: %w", err,
		)
	}

	result.SubmissionConditionsMet =
		playerStatus == "active" &&
			result.Status == QuestStatusActive &&
			result.HasRequiredEvidence &&
			result.OwnedItemCount >= result.RequiredItemCount

	return result, nil

}
