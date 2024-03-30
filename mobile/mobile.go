package mobile

import (
	"github.com/hajimehoshi/ebiten/v2/mobile"
	"github.com/patrick-me/game_one/game"
)

func init() {

	newGame, err := game.NewGame()
	if err != nil {
		panic(err)
	}

	// yourgame.Game must implement ebiten.Game interface.
	// For more details, see
	// * https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Game
	mobile.SetGame(newGame)
}

// Dummy is a dummy exported function.
//
// gomobile doesn't compile a package that doesn't include any exported function.
// Dummy forces gomobile to compile this package.
func Dummy() {}
