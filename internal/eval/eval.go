// internal/eval/eval.go
package eval

import (
	"abalone_go/internal/board"
	"container/list"
	"math"
)

/*
   启发指标
   ─────────
   h₁  中心距离差（越小越好）
   h₂  连通块数量差（块数越少越好）
   h₃  棋子数量差（子越多越好）

   附加特征
   ─────────
   hEdge  边缘惩罚
   hPush  潜在推子奖励
*/

// ─── 权重与阈值 ───
const (
	switchHigh     = 2.0
	switchLow      = 1.8
	centerFactor   = 1.0
	cohesionFactor = 1.0
	materialHigh   = 200.0 // 攻击权重↑
	materialMid    = 50.0

	edgeOne   = -50.0
	edgeZero  = -100.0
	pushBonus = 400.0 // 奖励↑
)

// Evaluate 计算 player 视角分数
func Evaluate(g *board.Game, player int8) int32 {
	opp := player ^ 1

	/* ---------- h₁ & hEdge ---------- */
	var dist, edge [2]float64
	for pos := int8(0); pos < board.N; pos++ {
		tok := g.TokenAt(pos)
		if tok != board.PlayerA && tok != board.PlayerB {
			continue
		}
		r, c := g.PosToCoord(pos)

		x := int8(c) - 5
		z := int8(r) - 5
		y := -x - z
		dist[tok] += float64((absI8(x) + absI8(y) + absI8(z)) / 2)

		if r == 1 || r == 9 || c == 1 || c == 9 {
			edge[tok] += edgeOne
		}
		if r == 0 || r == 10 || c == 0 || c == 10 {
			edge[tok] += edgeZero
		}
	}
	h1 := (dist[opp] - dist[player]) * centerFactor
	hEdge := edge[player] - edge[opp]

	/* ---------- h₂ ---------- */
	populations := func(p int8) int {
		vis := make([]bool, board.N)
		cnt := 0
		for pos := int8(0); pos < board.N; pos++ {
			if vis[pos] || g.TokenAt(pos) != p {
				continue
			}
			cnt++
			q := list.New()
			q.PushBack(pos)
			vis[pos] = true
			for q.Len() > 0 {
				cur := q.Remove(q.Front()).(int8)
				r, c := g.PosToCoord(cur)
				for _, d := range board.ACTIONS {
					rr, cc := r+d[0], c+d[1]
					nb := coordToPosSafe(g, rr, cc)
					if nb < 0 || vis[nb] || g.TokenAt(nb) != p {
						continue
					}
					vis[nb] = true
					q.PushBack(nb)
				}
			}
		}
		return cnt
	}
	h2 := float64(populations(player)-populations(opp)) * cohesionFactor

	/* ---------- h₃ ---------- */
	countPieces := func(p int8) int {
		n := 0
		for pos := int8(0); pos < board.N; pos++ {
			if g.TokenAt(pos) == p {
				n++
			}
		}
		return n
	}
	h3 := float64(countPieces(player) - countPieces(opp))

	/* ---------- hPush ---------- */
	hPush := potentialPush(g, player) - potentialPush(g, opp)

	/* ---------- 综合 ---------- */
	absH1 := math.Abs(h1 / centerFactor)
	switch {
	case absH1 > switchHigh:
		return int32(h1 + h2 + hEdge + hPush)
	case absH1 < switchLow:
		return int32(h1 + hEdge + hPush + h3*materialHigh)
	default:
		return int32(h1 + h2 + hEdge + hPush + h3*materialMid)
	}
}

/* ---------- 潜在推子检测 ---------- */

// coordToPosSafe: 越界返回 -1
func coordToPosSafe(g *board.Game, r, c int8) int8 {
	if r < 0 || r > 10 || c < 0 || c > 10 {
		return -1
	}
	return g.CoordToPos(r, c)
}

// potentialPush 检测 AAA E? □/VOID 型潜在推子
func potentialPush(g *board.Game, p int8) float64 {
	bonus := 0.0
	for pos := int8(0); pos < board.N; pos++ {
		if g.TokenAt(pos) != p {
			continue
		}
		r, c := g.PosToCoord(pos)
		for _, d := range board.ACTIONS {
			p1 := coordToPosSafe(g, r+d[0], c+d[1])
			p2 := coordToPosSafe(g, r+2*d[0], c+2*d[1])
			p3 := coordToPosSafe(g, r+3*d[0], c+3*d[1])
			p4 := coordToPosSafe(g, r+4*d[0], c+4*d[1])

			if p1 < 0 || p2 < 0 { // AAA 必须完整
				continue
			}
			if g.TokenAt(p1) != p || g.TokenAt(p2) != p {
				continue
			}
			// 第 3 格不能是己子或空格
			if p3 < 0 || g.TokenAt(p3) == p || g.TokenAt(p3) == board.TokenEmpty {
				continue
			}
			// 尾端允许越界(-1) / 空 / VOID
			if p4 >= 0 && g.TokenAt(p4) != board.TokenEmpty && g.TokenAt(p4) != board.TokenVoid {
				continue
			}
			bonus += pushBonus
		}
	}
	return bonus
}

/* ---------- 工具 ---------- */

func absI8(x int8) int8 {
	if x < 0 {
		return -x
	}
	return x
}
