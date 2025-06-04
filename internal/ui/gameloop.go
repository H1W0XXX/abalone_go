// File: internal/ui/gameloop.go
package ui

import (
	"abalone_go/internal/search"
	"log"
	"time"

	"abalone_go/internal/board"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	maxFPS  = 10
	screenW = 880
	screenH = 600 + 100
)

type GameLoop struct {
	logic  *board.Game
	rend   *renderer
	input  *inputHandler
	header *headerUI

	pve         bool // true=pve, false=pvp
	searchDepth int8
	humanSide   int8 // 仅 pve 有用
}

func NewGameLoop(g *board.Game, pve bool, depth int8) *GameLoop {
	return &GameLoop{
		logic: g,
		rend:  newRenderer(),
		input: &inputHandler{
			selPos:    -1,
			pvp:       !pve,          // pvp = 非 pve
			humanSide: board.PlayerA, // 白方为人（仅 PvE 用）
		},
		header:      newHeaderUI(),
		pve:         pve,
		searchDepth: depth,
		humanSide:   board.PlayerA,
	}
}

func (gl *GameLoop) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	// ① 轮到 AI 并且启用 PVE 模式
	if gl.pve && gl.logic.CurrentPlayer == board.PlayerB && !gl.logic.GameOver {
		best0, best1, _, _ := search.BestMoveParallel(gl.logic, gl.searchDepth, 15*time.Second)

		if ok, _, mods := gl.logic.ValidateMove(best0, best1); ok {
			gl.rend.applyModifications(mods, gl.logic)
			gl.logic.Apply(mods)
		}
		return nil
	}

	// ② 处理人类点击
	if mods := gl.input.handleMouse(gl.logic); mods != nil {
		gl.rend.applyModifications(mods, gl.logic)
		gl.logic.Apply(mods)
	}
	return nil
}

func (gl *GameLoop) Draw(screen *ebiten.Image) {
	gl.rend.drawBoard(screen, gl.logic, gl.input.selPos)
	gl.header.draw(screen, gl.logic)
}

func (gl *GameLoop) Layout(_, _ int) (int, int) { return screenW, screenH }

func Run(g *GameLoop) {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Abalone-Go (Ebiten)")

	// 限制到 10 FPS（10 次 Update+Draw 每秒）
	ebiten.SetTPS(maxFPS)
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
