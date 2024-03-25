package main

import (
	"github.com/gorilla/websocket"
	e "github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/patrick-me/game_one/game"

	"sort"
	"strconv"
)

var world *game.World
var frame int
var backgroundImg *e.Image
var imgPool map[string]*e.Image
var c *websocket.Conn

const (
	screenWidth  = 640
	screenHeight = 480
)

func init() {
	world = &game.World{
		IsServer: false,
		Units:    game.Units{},
	}

	backgroundImg, _, _ = ebitenutil.NewImageFromFile("resources/frames/bg.png", e.FilterDefault)
	imgPool = make(map[string]*e.Image)
	c = connectToServer()
}

type Game struct{}

func main() {

	e.SetRunnableOnUnfocused(true)
	e.SetWindowSize(screenWidth, screenHeight)
	e.SetWindowTitle("Game one")
	e.RunGame(&Game{})
	//e.Run()
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

func (g *Game) Draw(screen *e.Image) {
	g.Update(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) Update(screen *e.Image) error {
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
			img, _, _ = ebitenutil.NewImageFromFile(path, e.FilterDefault)
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

	if world.Units[world.MyID].Action == game.ActionRun {
		c.WriteJSON(game.Event{
			Type: game.EventTypeIdle,
			Data: game.EventMove{
				UnitID: world.MyID,
			},
		})
	}

	return nil
}
