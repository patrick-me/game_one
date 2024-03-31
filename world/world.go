package world

import (
	"github.com/google/uuid"
	_ "github.com/patrick-me/game_one/proto"
	events "github.com/patrick-me/game_one/proto"
	"math/rand"
	"time"
)

type World struct {
	MyID     string
	IsServer bool
	Units    map[string]*events.Unit
}

func (w *World) HandleEvent(e *events.Event) {
	switch e.Type {
	case events.Event_CONNECT:
		event := e.GetConnect()
		w.Units[event.Unit.ID] = event.Unit

	case events.Event_INIT:
		event := e.GetInit()
		if !w.IsServer {
			w.MyID = event.PlayerID
			w.Units = event.Units
		}

	case events.Event_MOVE:
		event := e.GetMove()
		unit := w.Units[event.UnitID]
		unit.Action = events.Action_RUN

		switch event.Direction {
		case events.Direction_UP:
			unit.Y--
		case events.Direction_DOWN:
			unit.Y++
		case events.Direction_LEFT:
			unit.X--
			unit.HorizontalDirection = event.Direction
		case events.Direction_RIGHT:
			unit.X++
			unit.HorizontalDirection = event.Direction

		}

	case events.Event_IDLE:
		event := e.GetIdle()

		unit := w.Units[event.UnitID]
		unit.Action = events.Action_IDLE

	case events.Event_DISCONNECT:
		event := e.GetDisconnect()
		delete(w.Units, event.UnitID)
	}

}

func (w *World) AddPlayer() *events.Unit {

	skins := []string{
		"elf_f", "elf_m", "knight_f", "knight_m", "lizard_f", "lizard_m", "wizzard_f", "wizzard_m",
	}
	id := uuid.New().String()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	unit := &events.Unit{
		ID:         id,
		Action:     events.Action_IDLE,
		X:          rnd.Float64() * 320,
		Y:          rnd.Float64() * 320,
		Frame:      int32(rnd.Intn(4)),
		SpriteName: skins[rnd.Intn(len(skins))],
	}

	w.Units[id] = unit
	return unit

}
