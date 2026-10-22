package handler

import (
	"cybertown/internal/game"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InventoryHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := game.GetInventory(
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

			log.Printf("get inventory: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to load inventory",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
		})
	}
}
