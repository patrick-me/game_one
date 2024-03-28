package main

import (
	"github.com/gin-gonic/gin"
	"github.com/patrick-me/game_one/game"
)

func main() {

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
	ws.Run(":3000")

}
