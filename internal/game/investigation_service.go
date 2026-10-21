package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

var (
	ErrQuestNotActive            = errors.New("quest is not active")
	ErrOutsideInvestigationRange = errors.New("player is outside investigation range")
)

type InvestigationResult struct {
	PointID    string `json:"point_id"`
	EvidenceID string `json:"evidence_id"`
	NewlyFound bool   `json:"newly_found"`
	Message    string `json:"message"`
}

func InspectAbandonedCart(
	ctx context.Context,
	db *sql.DB,
	worldID string,
	playerID string,
) (InvestigationResult, error) {
	if worldID == "" || playerID == "" {
		return InvestigationResult{}, fmt.Errorf(
			"world id and player id are required",
		)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return InvestigationResult{}, fmt.Errorf(
			"begin investigation: %w", err,
		)
	}
	defer tx.Rollback()

	// 在事务内读取玩家，后续查询和写入也使用 tx。
	var player Player

	err = tx.QueryRowContext(ctx, `
		SELECT scene_id, position_x, position_y, floor, status
		FROM players
		WHERE world_id = ? AND id = ?
	`, worldID, playerID).Scan(
		&player.SceneID,
		&player.Position.X,
		&player.Position.Y,
		&player.Floor,
		&player.Status,
	)
	if err != nil {
		return InvestigationResult{}, fmt.Errorf(
			"load investigation player: %w", err,
		)
	}

	if player.Status != "active" {
		return InvestigationResult{}, ErrPlayerInactive
	}

	// 必须已经接受货车任务，而且任务仍在进行中。
	var questStatus string

	err = tx.QueryRowContext(ctx, `
		SELECT status
		FROM player_quests
		WHERE world_id = ?
			AND player_id = ?
			AND quest_id = ?
	`, worldID, playerID, "missing_iron_cart").Scan(&questStatus)

	if errors.Is(err, sql.ErrNoRows) {
		return InvestigationResult{}, ErrQuestNotActive
	}
	if err != nil {
		return InvestigationResult{}, fmt.Errorf(
			"load investigation quest: %w", err,
		)
	}
	if questStatus != QuestStatusActive {
		return InvestigationResult{}, ErrQuestNotActive
	}

	// 对应 Godot 中 MissingCart 相对于 Game 的位置。
	cartPosition := Position{X: -5, Y: -96}
	const interactionDistance = 24.0

	if player.SceneID != "town" || player.Floor != 0 {
		return InvestigationResult{}, ErrOutsideInvestigationRange
	}

	distance := math.Hypot(
		player.Position.X-cartPosition.X,
		player.Position.Y-cartPosition.Y,
	)
	if math.IsNaN(distance) || math.IsInf(distance, 0) ||
		distance > interactionDistance {
		return InvestigationResult{}, ErrOutsideInvestigationRange
	}

	// 同一证据只保存一次，重复调查不覆盖首次发现记录。
	result, err := tx.ExecContext(ctx, `
		INSERT INTO player_evidence (
			world_id, player_id, evidence_id, source_id
		)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(world_id, player_id, evidence_id) DO NOTHING
	`,
		worldID,
		playerID,
		"cart_found",
		"abandoned_cart",
	)
	if err != nil {
		return InvestigationResult{}, fmt.Errorf(
			"save investigation evidence: %w", err,
		)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return InvestigationResult{}, fmt.Errorf(
			"read evidence insert result: %w", err,
		)
	}

	if err := tx.Commit(); err != nil {
		return InvestigationResult{}, fmt.Errorf(
			"commit investigation: %w", err,
		)
	}

	response := InvestigationResult{
		PointID:    "abandoned_cart",
		EvidenceID: "cart_found",
		NewlyFound: affected == 1,
		Message:    "已确认失踪货车的位置。",
	}
	if !response.NewlyFound {
		response.Message = "你已经调查过这里，货车位置已记录。"
	}

	return response, nil
}
