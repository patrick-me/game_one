package main

import (
	e "github.com/hajimehoshi/ebiten/v2"
	"github.com/patrick-me/game_one/game"
)

const (
	screenWidth  = 320
	screenHeight = 320
)

func main() {
	e.SetRunnableOnUnfocused(true)
	e.SetWindowSize(2*screenWidth, 2*screenHeight)
	e.SetWindowTitle("Game one")
	newGame, err := game.NewGame()
	if err != nil {
		panic(err)
	}
	e.RunGame(newGame)
}
