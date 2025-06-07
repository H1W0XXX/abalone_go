// File internal/ui/anim.go
package ui

import (
	"abalone_go/internal/board"
	"math"
	"time"
)

const moveDur = 300 * time.Millisecond // 动画时长

// pieceAnim 存储一次走子/推子的完整动画轨迹
type pieceAnim struct {
	piece    int8 // 直接用 int8，值是 board.Modification.Piece
	from     int8 // 起始 pos 索引
	to       int8 // 结束 pos 索引
	start    time.Time
	dirIndex int8 // 推出方向，0-5；如果普通移动可设 -1
	slotIdx  int8 // 如果被推出，落到右侧三角的第几个格 (0‥5)，否则 -1
}

// screenXY 直接在开始/结束像素坐标间线性插值
func (a *pieceAnim) screenXY(startXY, endXY func() (float64, float64)) (x, y float64, done bool) {
	elapsed := time.Since(a.start)
	t := float64(elapsed) / float64(moveDur)
	if t >= 1 {
		xx, yy := endXY()
		return xx, yy, true
	}
	p := math.Min(t, 1)
	x0, y0 := startXY()
	x1, y1 := endXY()
	return x0 + (x1-x0)*p, y0 + (y1-y0)*p, false
}

func (gl *GameLoop) startAnimations(mods []board.Modification) {
	// 1️⃣ 先让 renderer 记分，这时棋子仍在 OldPos 上
	gl.rend.applyModifications(mods, gl.logic) // outCounts 正确递增

	// 2️⃣ 计算每颗被推出棋子的槽位 （0‥5），保持与 outCounts 一致
	nextSlot := [2]int8{int8(outCounts[0] - 1), int8(outCounts[1] - 1)}
	gl.animating = make([]*pieceAnim, len(mods))

	for i, m := range mods {
		from, to := m.OldPos, m.NewPos
		color := gl.logic.TokenAt(from) // 现在还能取到真实颜色

		slot := int8(-1)
		if to == -1 { // 这颗棋子被推出
			nextSlot[color]++      // outCounts 刚才已经 +1，所以先自增
			slot = nextSlot[color] // 新棋子落到最新槽位 (0-based)
		}

		gl.animating[i] = &pieceAnim{
			piece:   color,
			from:    from,
			to:      to,
			slotIdx: slot,
			start:   time.Now(),
		}
	}

	// 3️⃣ 最后再真正修改棋盘
	gl.logic.Apply(mods)

	// 4️⃣ 锁输入
	gl.lockInput = true
}
