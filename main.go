package main

import (
	"github.com/gorilla/websocket"
	e "github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/patrick-me/game_one/game"
	"go.uber.org/zap"
	"sort"
	"strconv"
)

var world *game.World
var frame int
var backgroundImg *e.Image
var imgPool map[string]*e.Image
var c *websocket.Conn

const (
	screenWidth  = 320
	screenHeight = 320
)

var logger *zap.Logger

func init() {
	world = &game.World{
		IsServer: false,
		Units:    game.Units{},
	}

	backgroundImg, _, _ = ebitenutil.NewImageFromFile("resources/frames/bg.png")
	imgPool = make(map[string]*e.Image)
	c = connectToServer()

	logger, _ = zap.NewProduction()
	defer logger.Sync()
}

type Game struct{}

func main() {
	e.SetRunnableOnUnfocused(true)
	e.SetWindowSize(2*screenWidth, 2*screenHeight)
	e.SetWindowTitle("Game one")
	logger.Info("Running game")
	e.RunGame(&Game{})
}

func connectToServer() *websocket.Conn {
	c, _, _ := websocket.DefaultDialer.Dial("ws://localhost:3000/ws", nil)
	go func(c *websocket.Conn) {
		defer c.Close()

		for {
			var e game.Event
			c.ReadJSON(&e)
			world.HandleEvent(&e)
		}
	}(c)
	return c
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (_, _ int) {
	return screenWidth, screenHeight
}

func (g *Game) Draw(screen *e.Image) {
	frame++

	screen.DrawImage(backgroundImg, nil)
	unitList := []*game.Unit{}
	for _, unit := range world.Units {
		unitList = append(unitList, unit)
	}

	sort.Slice(unitList, func(i, j int) bool {
		return unitList[i].Y < unitList[j].Y
	})

	for _, unit := range unitList {
		spriteIndex := (frame/10 + unit.Frame) % 4
		op := &e.DrawImageOptions{}

		if unit.HorizontalDirection == game.DirectionLeft {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(16, 0)
		}

		op.GeoM.Translate(unit.X, unit.Y)

		path := "resources/frames/" + unit.SpriteName + "_" + unit.Action + "_anim_f" +
			strconv.Itoa(spriteIndex) + ".png"

		var img *e.Image
		var ok bool

		if img, ok = imgPool[path]; !ok {
			img, _, _ = ebitenutil.NewImageFromFile(path)
			imgPool[path] = img
		}

		screen.DrawImage(img, op)
	}

	if e.IsKeyPressed(e.KeyD) || e.IsKeyPressed(e.KeyRight) {
		c.WriteJSON(game.Event{
			Type: game.EventTypeMove,
			Data: game.EventMove{
				UnitID:    world.MyID,
				Direction: game.DirectionRight,
			},
		})
	}

	if e.IsKeyPressed(e.KeyA) || e.IsKeyPressed(e.KeyLeft) {
		c.WriteJSON(game.Event{
			Type: game.EventTypeMove,
			Data: game.EventMove{
				UnitID:    world.MyID,
				Direction: game.DirectionLeft,
			},
		})
	}

	if e.IsKeyPressed(e.KeyW) || e.IsKeyPressed(e.KeyUp) {
		c.WriteJSON(game.Event{
			Type: game.EventTypeMove,
			Data: game.EventMove{
				UnitID:    world.MyID,
				Direction: game.DirectionUp,
			},
		})
	}

	if e.IsKeyPressed(e.KeyS) || e.IsKeyPressed(e.KeyDown) {
		c.WriteJSON(game.Event{
			Type: game.EventTypeMove,
			Data: game.EventMove{
				UnitID:    world.MyID,
				Direction: game.DirectionDown,
			},
		})
	}

	unit, ok := world.Units[world.MyID]
	if ok && unit.Action == game.ActionRun {
		c.WriteJSON(game.Event{
			Type: game.EventTypeIdle,
			Data: game.EventMove{
				UnitID: world.MyID,
			},
		})
	}
}
