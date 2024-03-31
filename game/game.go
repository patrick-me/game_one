package game

import (
	"github.com/gorilla/websocket"
	e "github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/joho/godotenv"
	_ "github.com/patrick-me/game_one/proto"
	events "github.com/patrick-me/game_one/proto"
	w "github.com/patrick-me/game_one/world"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

type Game struct {
	ScreenWidth   int
	ScreenHeight  int
	Frame         int
	World         *w.World
	Conn          *websocket.Conn
	BackgroundImg *e.Image
	ImgPool       map[string]*e.Image
}

var world *w.World
var frame int
var backgroundImg *e.Image
var imgPool map[string]*e.Image
var c *websocket.Conn
var logger *zap.Logger

func init() {
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found")
	}

	world = &w.World{
		IsServer: false,
		Units:    make(map[string]*events.Unit),
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
			_, m, err2 := cn.ReadMessage()
			if err2 != nil {
				logger.Info("can't read event", zap.Error(err))
			}
			var event events.Event
			proto.Unmarshal(m, &event)
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
		sendEvent(g, events.Direction_RIGHT)
		return nil
	}

	if e.IsKeyPressed(e.KeyA) || e.IsKeyPressed(e.KeyLeft) {
		sendEvent(g, events.Direction_LEFT)
		return nil
	}

	if e.IsKeyPressed(e.KeyW) || e.IsKeyPressed(e.KeyUp) {
		sendEvent(g, events.Direction_UP)
		return nil
	}

	if e.IsKeyPressed(e.KeyS) || e.IsKeyPressed(e.KeyDown) {
		sendEvent(g, events.Direction_DOWN)
		return nil
	}

	unit, ok := g.World.Units[g.World.MyID]
	if ok && unit.Action == events.Action_RUN {
		event := events.Event{
			Type: events.Event_IDLE,
			Data: &events.Event_Idle{
				Idle: &events.EventIdle{
					UnitID: g.World.MyID,
				},
			},
		}
		msg, _ := proto.Marshal(&event)
		g.Conn.WriteMessage(websocket.BinaryMessage, msg)
		return nil
	}

	return nil
}

func sendEvent(g *Game, direction events.Direction) {
	event := events.Event{
		Type: events.Event_MOVE,
		Data: &events.Event_Move{
			Move: &events.EventMove{
				UnitID:    g.World.MyID,
				Direction: direction,
			},
		},
	}
	msg, _ := proto.Marshal(&event)
	g.Conn.WriteMessage(websocket.BinaryMessage, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (_, _ int) {
	return g.ScreenWidth, g.ScreenHeight
}

func (g *Game) Draw(screen *e.Image) {
	g.Frame++

	screen.DrawImage(g.BackgroundImg, nil)
	unitList := []*events.Unit{}
	for _, unit := range g.World.Units {
		unitList = append(unitList, unit)
	}

	sort.Slice(unitList, func(i, j int) bool {
		return unitList[i].Y < unitList[j].Y
	})

	for _, unit := range unitList {
		spriteIndex := (int(g.Frame)/8 + int(unit.Frame)) % 4
		op := &e.DrawImageOptions{}

		if unit.HorizontalDirection == events.Direction_LEFT {
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

		var a string
		switch unit.Action {
		case events.Action_RUN:
			a = "run"
		case events.Action_IDLE:
			a = "idle"

		}
		path := "resources/frames/" + unit.SpriteName + "_" + a + "_anim_f" +
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
