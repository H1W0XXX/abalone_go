// File: internal/ui/gameloop.go
package ui

import (
	"abalone_go/internal/board"
	"abalone_go/internal/search"
	"github.com/hajimehoshi/ebiten/v2"
	"log"
	"time"
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

	animating []*pieceAnim
	lockInput bool
}

// posToXY 把格子索引 pos 转为屏幕像素坐标
func posToXY(pos int8) (float64, float64) {
	cx := cellCenters[pos][0]
	cy := cellCenters[pos][1]
	return float64(cx), float64(cy)
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
	// ① Esc 退出
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	// ② 动画阶段：全速
	if len(gl.animating) > 0 {
		leavePerf() // 高性能模式（不省电）

		allDone := true
		for _, a := range gl.animating {
			startXY := func() (float64, float64) {
				c := cellCenters[int(a.from)]
				return float64(c[0] - 24), float64(c[1] - 24)
			}
			endXY := func() (float64, float64) {
				if a.to >= 0 {
					c := cellCenters[int(a.to)]
					return float64(c[0] - 24), float64(c[1] - 24)
				}
				idx := a.slotIdx
				oc := outCoords[int(a.piece)][idx]
				return float64(oc[0] - 24), float64(oc[1] - 24)
			}
			_, _, done := a.screenXY(startXY, endXY)
			if !done {
				allDone = false
			}
		}
		if allDone {
			gl.animating = nil
			gl.lockInput = false

		} else {
			gl.lockInput = true
		}
		return nil
	}

	// ③ AI 回合：这时通常不需要高帧率（省电即可）
	if gl.pve && gl.logic.CurrentPlayer == board.PlayerB && !gl.logic.GameOver {
		// （注意：最好不要在 Update 里做长时间阻塞搜索，建议用 goroutine + 标志位。
		// 但若你现在就是同步搜索，也不必切离省电。）
		best0, best1, _, _ := search.BestMoveParallel(gl.logic, gl.searchDepth, 15*time.Second)
		if ok, _, mods := gl.logic.ValidateMove(best0, best1); ok {
			gl.startAnimations(mods)
		}

		return nil
	}

	// ④ 玩家输入：省电状态下也能响应；一旦要播动画再切全速
	if mods := gl.input.handleMouse(gl.logic, gl.lockInput); mods != nil {

		gl.startAnimations(mods)
		return nil
	}

	return nil
}
func (gl *GameLoop) Draw(screen *ebiten.Image) {
	// 传入 gl 本身，让 drawBoard 能访问 gl.logic、gl.animating、gl.input.selPos
	gl.rend.drawBoard(screen, gl)
	gl.header.draw(screen, gl.logic)
}
func (gl *GameLoop) Layout(_, _ int) (int, int) { return screenW, screenH }

func Run(g *GameLoop) {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Abalone-Go (Ebiten)")

	// 限制到 10 FPS（10 次 Update+Draw 每秒）
	//ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMinimum)
	ebiten.SetTPS(maxFPS)
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
