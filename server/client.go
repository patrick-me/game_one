// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"github.com/patrick-me/game_one/game"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump(world *game.World, unitID string) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		removeDisconnectedUnit(c.hub, world, unitID)
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.hub.broadcast <- message

		var e game.Event
		err = json.Unmarshal(message, &e)
		if err != nil {
			logger.Error("can't unmarshal event",
				zap.String("message", string(message)),
				zap.Error(err))
		}
		world.HandleEvent(&e, logger)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(world *game.World, unitID string) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		removeDisconnectedUnit(c.hub, world, unitID)
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func ServeWs(hub *Hub, world *game.World, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("can't upgrade connection", zap.Error(err))
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	hub.register <- client

	player := sendToNewPlayerWorldUnits(world, conn)
	sendAllNewUnitConnected(hub, world, player)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump(world, player.ID)
	go client.readPump(world, player.ID)
}

func sendAllNewUnitConnected(hub *Hub, world *game.World, player *game.Unit) {
	event := game.Event{
		Type: game.EventTypeConnect,
		Data: game.EventConnect{
			Unit: *world.Units[player.ID],
		},
	}

	msg, _ := json.Marshal(event)

	hub.broadcast <- msg
}

func sendToNewPlayerWorldUnits(world *game.World, conn *websocket.Conn) *game.Unit {
	player := world.AddPlayer()
	event := game.Event{
		Type: game.EventTypeInit,
		Data: game.EventInit{
			PlayerID: player.ID,
			Units:    world.Units,
		},
	}
	logger.Info("New player added",
		zap.String("player", player.ID),
		zap.Int("units", len(world.Units)))

	conn.WriteJSON(event)
	return player
}

func removeDisconnectedUnit(hub *Hub, world *game.World, unitID string) {

	logger.Info("removing disconnected unit",
		zap.String("unitId", unitID))

	event := game.Event{
		Type: game.EventTypeUnitDisconnected,
		Data: game.EventUnitDisconnected{
			UnitID: unitID,
		},
	}

	msg, _ := json.Marshal(event)
	hub.broadcast <- msg
	delete(world.Units, unitID)
}
