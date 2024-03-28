package game

import (
	"encoding/json"
	"github.com/google/uuid"
	"math/rand"
	"time"
)

type Unit struct {
	ID                  string    `json:"id"`
	X                   float64   `json:"x"`
	Y                   float64   `json:"y"`
	SpriteName          string    `json:"sprite_name"`
	Action              string    `json:"action"`
	LastActionTime      time.Time `json:"last_action"`
	Frame               int       `json:"frame"`
	HorizontalDirection int       `json:"horizontal_direction"`
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
		unit.LastActionTime = time.Now()

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
		ID:             id,
		Action:         ActionIdle,
		LastActionTime: time.Now(),
		X:              rnd.Float64() * 320,
		Y:              rnd.Float64() * 240,
		Frame:          rnd.Intn(4),
		SpriteName:     skins[rnd.Intn(len(skins))],
	}

	w.Units[id] = unit
	return unit

}
