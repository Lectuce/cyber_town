package rouster

import (
	"cybertown/internal/game"
	"cybertown/internal/npc"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func SetUpRousters(r *gin.Engine, db *sql.DB, dialogueService *npc.DialogueService) {

	r.GET("/api/player", func(c *gin.Context) {

		player, err := game.GetPlayer(c.Request.Context(), db, "town-1", "player-1")

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
	})

	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	var chatMu sync.Mutex
	var lastChat time.Time

	r.POST("/api/npcs/:npcID/chat", func(c *gin.Context) {
		var body struct {
			RequestID string `json:"request_id" binding:"required"`
			Message   string `json:"message" binding:"required"`
		}

		c.Request.Body = http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			4096,
		)

		err := c.ShouldBindJSON(&body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid chat request",
			})
			return
		}

		chatMu.Lock()
		tooSoon := time.Since(lastChat) < 2*time.Second
		if !tooSoon {
			lastChat = time.Now()
		}
		chatMu.Unlock()

		if tooSoon {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "please wait before sending another message",
			})
			return
		}

		result, err := dialogueService.Chat(
			c.Request.Context(),
			npc.ChatRequest{
				WorldID:   "town-1",
				PlayerID:  "player-1",
				NPCID:     c.Param("npcID"),
				RequestID: body.RequestID,
				Message:   body.Message,
			},
		)
		if err != nil {
			switch {
			case errors.Is(err, npc.ErrInvalidRequest),
				errors.Is(err, npc.ErrInvalidMessage):
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})

			case errors.Is(err, npc.ErrDialogueBusy),
				errors.Is(err, npc.ErrRequestConflict):
				c.JSON(http.StatusConflict, gin.H{
					"error": err.Error(),
				})

			case errors.Is(err, npc.ErrPlayerInactive):
				c.JSON(http.StatusForbidden, gin.H{
					"error": "player is not active",
				})

			case errors.Is(err, sql.ErrNoRows):
				c.JSON(http.StatusNotFound, gin.H{
					"error": "player or NPC not found",
				})

			default:
				log.Printf("NPC chat failed: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "NPC dialogue failed",
				})
			}
			return
		}

		c.JSON(http.StatusOK, result)
	})

	log.Println("game server listening on http://127.0.0.1:8080")

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server stopped: %v", err)
	}

}
