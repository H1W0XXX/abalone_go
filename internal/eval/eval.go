// File: internal/eval/eval.go

package eval

import (
	"abalone_go/internal/board"
)

// ───────────────── 权重表 ─────────────────

// 预先为 61 个格子给定“越靠中心越高”权重，范围 0…40
// 下面这组手动填入：中心 0→40 依次递减；可根据实际棋盘坐标微调。
var centerWeight = [61]int8{
	40, 38, 38, 36, 36,
	34, 34, 34, 32, 32, 32,
	30, 30, 30, 30, 28, 28, 28,
	26, 26, 26, 26, 26, 24, 24,
	22, 22, 22, 22, 22, 20, 20,
	18, 18, 18, 18, 18, 16, 16,
	14, 14, 14, 14, 12, 12, 12,
	10, 10, 10, 8, 8, 6,
	4, 4, 2, 2, 0, 0, 0,
}

// ───────────────── 评估函数 ─────────────────

// Evaluate 返回 player 视角下局面分数（正=好）
func Evaluate(g *board.Game, player int8) int32 {
	var (
		myMaterial, oppMaterial int32
		myCenter, oppCenter     int32
		myConn, oppConn         int32
		myFront, oppFront       int8 = 8, 0 // row 最大/最小
	)

	// 遍历 61 格
	for pos := int8(0); pos < 61; pos++ {
		token := g.TokenAt(pos)
		if token != board.PlayerA && token != board.PlayerB {
			continue
		}
		r, _ := g.PosToCoord(pos) // 只用 r 判断前线
		if token == player {
			myMaterial++
			myCenter += int32(centerWeight[pos])
			if r < myFront {
				myFront = r
			}
		} else {
			oppMaterial++
			oppCenter += int32(centerWeight[pos])
			if r > oppFront {
				oppFront = r
			}
		}
		// 计算相邻友子
		for _, d := range board.ACTIONS {
			r, c := g.PosToCoord(pos)
			rr, cc := r+d[0], c+d[1]
			nbPos := g.CoordToPos(rr, cc)
			if nbPos < 0 {
				continue
			}
			if g.TokenAt(nbPos) == token {
				if token == player {
					myConn++
				} else {
					oppConn++
				}
			}
		}
	}

	score := int32(0)
	// 1. 材料
	score += (myMaterial - oppMaterial) * 1000
	// 2. 中心
	score += (myCenter - oppCenter) * 10
	// 3. 邻接
	score += (myConn - oppConn) * 30
	// 4. 前线（行号越小越靠敌人）
	score += int32(oppFront-myFront) * 20

	return score
}
