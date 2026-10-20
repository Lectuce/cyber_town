package handler

import (
	"cybertown/internal/game"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 查询任务介绍和当前玩家进度。
func QuestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		questID := c.Param("questID")

		definition, err := game.GetQuestDefinition(questID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "quest not found",
			})
			return
		}

		// 确认玩家存在
		_, err = game.GetPlayer(ctx, db, "town-1", "player-1")
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "player not found",
				})
				return
			}

			log.Printf("load quest player: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to load player",
			})
			return
		}

		quest, err := game.GetPlayerQuest(ctx, db, "town-1", "player-1", questID)

		status := "not_started"
		if err == nil {
			status = quest.Status
		} else if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("get player quest: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to load quest",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"quest_id":     definition.ID,
			"title":        definition.Title,
			"description":  definition.Description,
			"giver_npc_id": definition.GiverNPCID,
			"status":       status,
		})
	}
}

// 接受任务，保存玩家进度。
func AcceptQuestHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		questID := c.Param("questID")

		// 判断存不存在任务，返回明确的 404。
		_, err := game.GetQuestDefinition(questID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "quest not found",
			})
			return
		}

		quest, err := game.AcceptQuest(
			c.Request.Context(),
			db,
			"town-1",
			"player-1",
			questID,
		)
		if err != nil {
			switch {
			case errors.Is(err, game.ErrPlayerInactive):
				c.JSON(http.StatusForbidden, gin.H{
					"error": "player is not active",
				})

			case errors.Is(err, sql.ErrNoRows):
				c.JSON(http.StatusNotFound, gin.H{
					"error": "player or quest giver not found",
				})

			default:
				log.Printf("accept quest: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to accept quest",
				})
			}
			return
		}

		c.JSON(http.StatusOK, quest)
	}
}
