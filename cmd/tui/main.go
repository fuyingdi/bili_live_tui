package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shr-go/bili_live_tui/pkg/logging"
)

func main() {
	logging.Infof("Starting Web Server")

	r := gin.Default()

	// Serve static files (frontend)
	r.Static("/static", "./frontend")

	// API endpoint for sending danmu
	r.POST("/api/send", func(c *gin.Context) {
		var req struct {
			Message string `json:"message"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Simulate sending danmu
		logging.Infof("Received danmu: %s", req.Message)
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// WebSocket endpoint for real-time danmu
	r.GET("/ws", func(c *gin.Context) {
		// Placeholder for WebSocket implementation
		c.JSON(http.StatusOK, gin.H{"message": "WebSocket endpoint not implemented yet"})
	})

	// Start the server
	r.Run(":8080")
}
