package game

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	e "github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

type Unit struct {
	ID                  string  `json:"id"`
	X                   float64 `json:"x"`
	Y                   float64 `json:"y"`
	SpriteName          string  `json:"sprite_name"`
	Action              string  `json:"action"`
	Frame               int     `json:"frame"`
	HorizontalDirection int     `json:"horizontal_direction"`
}

type Units map[string]*Unit

type World struct {
	MyID     string `json:"-"`
	IsServer bool   `json:"-"`
	Units    `json:"units"`
}

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type EventConnect struct {
	Unit
}

type EventMove struct {
	UnitID    string `json:"unit_id"`
	Direction int    `json:"direction"`
}

type EventUnitDisconnected struct {
	UnitID string `json:"unit_id"`
}

type EventIdle struct {
	UnitID string `json:"unit_id"`
}

type EventInit struct {
	PlayerID string `json:"player_id"`
	Units    Units  `json:"units"`
}

const (
	EventTypeConnect          = "connect"
	EventTypeMove             = "move"
	EventTypeIdle             = "idle"
	EventTypeInit             = "init"
	EventTypeUnitDisconnected = "disconnect"
)

const ActionRun = "run"
const ActionIdle = "idle"

const DirectionUp = 0
const DirectionDown = 1
const DirectionLeft = 2
const DirectionRight = 3

func (w *World) HandleEvent(e *Event) {
	switch e.Type {
	case EventTypeConnect:
		str, _ := json.Marshal(e.Data)
		var event EventConnect
		json.Unmarshal(str, &event)

		w.Units[event.ID] = &event.Unit

	case EventTypeInit:
		str, _ := json.Marshal(e.Data)
		var event EventInit
		json.Unmarshal(str, &event)

		if !w.IsServer {
			w.MyID = event.PlayerID
			w.Units = event.Units
		}

	case EventTypeMove:
		str, _ := json.Marshal(e.Data)
		var event EventMove
		json.Unmarshal(str, &event)

		unit := w.Units[event.UnitID]
		unit.Action = ActionRun

		switch event.Direction {
		case DirectionUp:
			unit.Y--
		case DirectionDown:
			unit.Y++
		case DirectionLeft:
			unit.X--
			unit.HorizontalDirection = event.Direction
		case DirectionRight:
			unit.X++
			unit.HorizontalDirection = event.Direction

		}

	case EventTypeIdle:
		str, _ := json.Marshal(e.Data)
		var event EventIdle
		json.Unmarshal(str, &event)

		unit := w.Units[event.UnitID]
		unit.Action = ActionIdle

	case EventTypeUnitDisconnected:
		str, _ := json.Marshal(e.Data)
		var event EventUnitDisconnected
		json.Unmarshal(str, &event)
		delete(w.Units, event.UnitID)
	}

}

func (w *World) AddPlayer() *Unit {

	skins := []string{
		"elf_f", "elf_m", "knight_f", "knight_m", "lizard_f", "lizard_m", "wizzard_f", "wizzard_m",
	}
	id := uuid.New().String()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	unit := &Unit{
		ID:         id,
		Action:     ActionIdle,
		X:          rnd.Float64() * 320,
		Y:          rnd.Float64() * 320,
		Frame:      rnd.Intn(4),
		SpriteName: skins[rnd.Intn(len(skins))],
	}

	w.Units[id] = unit
	return unit

}

type Game struct {
	ScreenWidth   int
	ScreenHeight  int
	Frame         int
	World         *World
	Conn          *websocket.Conn
	BackgroundImg *e.Image
	ImgPool       map[string]*e.Image
}

var world *World
var frame int
var backgroundImg *e.Image
var imgPool map[string]*e.Image
var c *websocket.Conn
var logger *zap.Logger

func init() {
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found")
	}

	world = &World{
		IsServer: false,
		Units:    Units{},
	}

	backgroundImg, _, _ = ebitenutil.NewImageFromFile("resources/frames/bg.png")
	imgPool = make(map[string]*e.Image)

	c = connectToServer()

	logger, _ = zap.NewProduction()
	defer logger.Sync()

}

func connectToServer() *websocket.Conn {
	header := http.Header{}
	header.Set("Authorization", os.Getenv("AUTH_TOKEN"))
	conn, _, err := websocket.DefaultDialer.Dial(os.Getenv("CONNECTION_URL"), header)

	if err != nil {
		logger.Info("can't connect to server", zap.Error(err))
	}

	go func(cn *websocket.Conn) {
		defer cn.Close()

		for {
			var event Event
			err2 := cn.ReadJSON(&event)
			if err2 != nil {
				logger.Info("can't read event", zap.Error(err))
			}
			world.HandleEvent(&event)
		}
	}(conn)

	return conn
}

func NewGame() (*Game, error) {

	return &Game{
		ScreenWidth:   320,
		ScreenHeight:  320,
		Frame:         frame,
		World:         world,
		BackgroundImg: backgroundImg,
		ImgPool:       imgPool,
		Conn:          c,
	}, nil
}

func (g *Game) Update() error {
	if e.IsKeyPressed(e.KeyD) || e.IsKeyPressed(e.KeyRight) {
		sendEvent(g, DirectionRight)
		return nil
	}

	if e.IsKeyPressed(e.KeyA) || e.IsKeyPressed(e.KeyLeft) {
		sendEvent(g, DirectionLeft)
		return nil
	}

	if e.IsKeyPressed(e.KeyW) || e.IsKeyPressed(e.KeyUp) {
		sendEvent(g, DirectionUp)
		return nil
	}

	if e.IsKeyPressed(e.KeyS) || e.IsKeyPressed(e.KeyDown) {
		sendEvent(g, DirectionDown)
		return nil
	}

	unit, ok := g.World.Units[g.World.MyID]
	if ok && unit.Action == ActionRun {
		g.Conn.WriteJSON(Event{
			Type: EventTypeIdle,
			Data: EventIdle{
				UnitID: g.World.MyID,
			},
		})
		return nil
	}

	return nil
}

func sendEvent(g *Game, direction int) {
	g.Conn.WriteJSON(Event{
		Type: EventTypeMove,
		Data: EventMove{
			UnitID:    g.World.MyID,
			Direction: direction,
		},
	})
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (_, _ int) {
	return g.ScreenWidth, g.ScreenHeight
}

func (g *Game) Draw(screen *e.Image) {
	g.Frame++

	screen.DrawImage(g.BackgroundImg, nil)
	unitList := []*Unit{}
	for _, unit := range g.World.Units {
		unitList = append(unitList, unit)
	}

	sort.Slice(unitList, func(i, j int) bool {
		return unitList[i].Y < unitList[j].Y
	})

	for _, unit := range unitList {
		spriteIndex := (g.Frame/8 + unit.Frame) % 4
		op := &e.DrawImageOptions{}

		if unit.HorizontalDirection == DirectionLeft {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(16, 0)
		}

		op.GeoM.Translate(unit.X, unit.Y)
		second := time.Now().Second()

		if second >= 0 && second <= 2 {
			logger.Info("Position of unit: ",
				zap.String("unit", unit.ID),
				zap.Float64("X", unit.X),
				zap.Float64("Y", unit.Y),
			)
		}

		path := "resources/frames/" + unit.SpriteName + "_" + unit.Action + "_anim_f" +
			strconv.Itoa(spriteIndex) + ".png"

		var img *e.Image
		var ok bool

		if img, ok = g.ImgPool[path]; !ok {
			img, _, _ = ebitenutil.NewImageFromFile(path)
			g.ImgPool[path] = img
		}

		screen.DrawImage(img, op)
	}
}
