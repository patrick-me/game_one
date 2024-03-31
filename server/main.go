package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	events "github.com/patrick-me/game_one/proto"
	w "github.com/patrick-me/game_one/world"
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

	world := &w.World{
		IsServer: true,
		Units:    make(map[string]*events.Unit),
	}

	hub := NewHub()
	go hub.run()

	ws := gin.New()
	ws.GET("/ws", wsHandler(hub, world))

	ticker := time.NewTicker(time.Hour * 1)
	done := make(chan bool)

	go worldInfo(done, ticker, world)

	logger.Info("Listening on port: ", zap.String("port", os.Getenv("SERVER_PORT")))
	ws.Run(":" + os.Getenv("SERVER_PORT"))

	ticker.Stop()
	done <- true
}

func worldInfo(done chan bool, ticker *time.Ticker, world *w.World) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			logger.Info("units in the world", zap.Int("units", len(world.Units)))
		}
	}
}

func wsHandler(hub *Hub, world *w.World) gin.HandlerFunc {
	return func(hub *Hub, world *w.World) gin.HandlerFunc {
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
