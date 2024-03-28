package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/patrick-me/game_one/game"
	"go.uber.org/zap"
	"os"
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
	ws.GET("/ws", func(hub *Hub, world *game.World) gin.HandlerFunc {
		return func(c *gin.Context) {
			ServeWs(hub, world, c.Writer, c.Request)
		}
	}(hub, world))

	logger.Info("Listening on port: ", zap.String("port", os.Getenv("SERVER_PORT")))
	ws.Run(":" + os.Getenv("SERVER_PORT"))
}
