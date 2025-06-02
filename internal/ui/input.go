package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"abalone_go/internal/board"
)

type inputHandler struct {
	selPos int8 // -1 表示没有选中
}

// handleMouse 处理点击；合法走子时返回 mods，否则返回 nil
func (h *inputHandler) handleMouse(g *board.Game) []board.Modification {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	x, y := ebiten.CursorPosition()
	pos := pixelToPos(x, y)
	if pos < 0 {
		return nil
	}

	token := g.TokenAt(pos)
	isOwn := token == g.CurrentPlayer

	// 第一次选中
	if h.selPos == -1 {
		if isOwn {
			h.selPos = pos
		}
		return nil
	}

	// 已有选中，再点己方 -> 切换选中
	if isOwn {
		h.selPos = pos
		return nil
	}

	// 否则尝试走子
	ok, _, mods := g.ValidateMove(h.selPos, pos)
	h.selPos = -1 // 清空选中
	if ok {
		return mods
	}
	return nil
}

/* ---------- 像素坐标 -> 格子索引 ---------- */

func pixelToPos(x, y int) int8 {
	best := int8(-1)
	bestDist := 24 * 24
	for p := int8(0); p < 61; p++ {
		dx := x - cellCenters[p][0]
		dy := y - cellCenters[p][1]
		d2 := dx*dx + dy*dy
		if d2 < bestDist {
			bestDist, best = d2, p
		}
	}
	return best
}
