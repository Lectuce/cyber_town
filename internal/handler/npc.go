package handler

import (
	"cybertown/internal/npc"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func NPCChatHandler(dialogueService *npc.DialogueService) gin.HandlerFunc {
	var chatMu sync.Mutex
	var lastChat time.Time

	return func(c *gin.Context) {
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

		isStream := c.FullPath() == "/api/npcs/:npcID/chat/stream"
		streamStarted := false

		// 向浏览器发送一个完整的 SSE 事件。
		sendEvent := func(event string, payload any) error {
			if err := c.Request.Context().Err(); err != nil {
				return err
			}

			data, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			if !streamStarted {
				c.Header("Content-Type", "text/event-stream; charset=utf-8")
				c.Header("Cache-Control", "no-cache, no-transform")
				c.Header("X-Accel-Buffering", "no")
				streamStarted = true
			}

			_, err = fmt.Fprintf(
				c.Writer,
				"event: %s\ndata: %s\n\n",
				event,
				data,
			)
			if err != nil {
				return err
			}

			// 立即把本段数据交给底层连接。
			c.Writer.Flush()
			return nil
		}

		var onText func(string) error

		if isStream {
			onText = func(text string) error {
				return sendEvent("delta", gin.H{
					"text": text,
				})
			}
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
			onText,
		)

		if err != nil {
			// 已经开始发送流，就不能再切换成普通 JSON 错误响应。
			if streamStarted {
				log.Printf("NPC stream failed: %v", err)

				sendErr := sendEvent("error", gin.H{
					"request_id": body.RequestID,
					"error":      "NPC dialogue failed",
				})
				if sendErr != nil {
					log.Printf("send stream error event: %v", sendErr)
				}

				return
			}

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

		if isStream {
			// Chat 返回成功，代表整轮生成和数据库保存都已成功。
			err := sendEvent("done", result)
			if err != nil {
				log.Printf("send stream completion: %v", err)
			}
			return
		}

		c.JSON(http.StatusOK, result)

	}

}
