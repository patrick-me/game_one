package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/patrick-me/game_one/game"
	"go.uber.org/zap"
	"os"
	"time"
)

var logger *zap.Logger

func init() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()

	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found")
	}
}

func main() {
	defer logger.Sync()

	world := &game.World{
		IsServer: true,
		Units:    game.Units{},
	}

	hub := NewHub()
	go hub.run()

	ws := gin.New()
	ws.GET("/ws", wsHandler(hub, world))

	ticker := time.NewTicker(time.Minute * 1)
	done := make(chan bool)

	go clearWorld(done, ticker, world)

	logger.Info("Listening on port: ", zap.String("port", os.Getenv("SERVER_PORT")))
	ws.Run(":" + os.Getenv("SERVER_PORT"))

	ticker.Stop()
	done <- true
}

func clearWorld(done chan bool, ticker *time.Ticker, world *game.World) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			logger.Info("units in the world", zap.Int("units", len(world.Units)))
		}
	}
}

func wsHandler(hub *Hub, world *game.World) gin.HandlerFunc {
	return func(hub *Hub, world *game.World) gin.HandlerFunc {
		return func(c *gin.Context) {
			auth := c.Request.Header.Get("Authorization")

			if auth != os.Getenv("AUTH_TOKEN") {
				logger.Info("Request without authorization", zap.String("auth", auth))
				return
			}
			ServeWs(hub, world, c.Writer, c.Request)
		}
	}(hub, world)
}
