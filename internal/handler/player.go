package handler

import (
	"cybertown/internal/game"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PlayerHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		player, err := game.GetPlayer(
			c.Request.Context(),
			db,
			"town-1",
			"player-1",
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "player not found",
				})
				return
			}

			log.Printf("get player: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to load player",
			})
			return
		}

		c.JSON(http.StatusOK, player)
	}
}
