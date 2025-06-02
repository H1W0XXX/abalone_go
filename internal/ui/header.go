package ui

import (
	"fmt"
	"image/color"

	"abalone_go/internal/board"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

type headerUI struct{}

func newHeaderUI() *headerUI { return &headerUI{} }

var colWhite = color.White

func (h *headerUI) draw(screen *ebiten.Image, g *board.Game) {
	y := 600 + 50 // header 垂直居中
	x := 10

	strs := []string{
		fmt.Sprintf("Player | %d", g.CurrentPlayer),
		fmt.Sprintf("Episode | %d", g.PlayerVictories[0]+g.PlayerVictories[1]+1),
		fmt.Sprintf("Turns | %d", g.TurnCount),
		fmt.Sprintf("State | %s", map[bool]string{false: "ON GOING", true: "OVER"}[g.GameOver]),
		fmt.Sprintf("Score | A:%d  B:%d", g.PlayerVictories[0], g.PlayerVictories[1]),
	}
	for _, s := range strs {
		text.Draw(screen, s, basicfont.Face7x13, x, y, colWhite)
		x += len(s)*7 + 30
	}
}
