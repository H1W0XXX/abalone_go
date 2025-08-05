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

	// ② 动画阶段 ────────────────────────────
	if len(gl.animating) > 0 {
		allDone := true

		for _, a := range gl.animating {
			// 起点闭包
			startXY := func() (float64, float64) {
				c := cellCenters[int(a.from)] // ← 记得转 int
				return float64(c[0] - 24), float64(c[1] - 24)
			}
			// 终点闭包
			endXY := func() (float64, float64) {
				if a.to >= 0 { // 普通落子
					c := cellCenters[int(a.to)]
					return float64(c[0] - 24), float64(c[1] - 24)
				}
				// 推子：用 slotIdx 做索引
				idx := a.slotIdx // 0‥5
				oc := outCoords[int(a.piece)][idx]
				return float64(oc[0] - 24), float64(oc[1] - 24)
			}

			_, _, done := a.screenXY(startXY, endXY)
			if !done {
				allDone = false
			}
		}

		// 全部动画播完 ⇒ 清空并解锁
		if allDone {
			gl.animating = nil
		}
		gl.lockInput = len(gl.animating) > 0 // 下一帧是否允许点击
		return nil
	}

	// ③ AI 走子（PvE 且轮到黑方） ────────────
	if gl.pve && gl.logic.CurrentPlayer == board.PlayerB && !gl.logic.GameOver {
		best0, best1, _, _ := search.BestMoveParallel(gl.logic, gl.searchDepth, 15*time.Second)
		if ok, _, mods := gl.logic.ValidateMove(best0, best1); ok {
			gl.startAnimations(mods) // 会把 lockInput 设为 true
		}
		return nil
	}

	// ④ 玩家点击 ─────────────────────────────
	if mods := gl.input.handleMouse(gl.logic, gl.lockInput); mods != nil {
		gl.startAnimations(mods) // 同样开启动画并锁输入
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
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMinimum)
	ebiten.SetTPS(maxFPS)
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
