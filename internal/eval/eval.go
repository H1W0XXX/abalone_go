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
	switchLow      = 3.1
	centerFactor   = 1.0
	cohesionFactor = 1.0
	materialHigh   = 200.0 // 攻击权重↑
	materialMid    = 50.0

	edgeOne           = -50.0
	edgeZero          = -100.0
	edgePenaltyStrong = -600.0 // 贴边即罚
	pushBonus         = 350.0  // 略收敛
	capturedBonus     = 5000.0 // 吃子权重，可自行调节

)

// edgePenalty 返回 p 方“孤立”贴边子惩罚
func edgePenalty(g *board.Game, p int8) float64 {
	bad := 0.0
	for pos := int8(0); pos < board.N; pos++ {
		if g.TokenAt(pos) != p {
			continue
		}
		// 若该子与同向己子相连 => 可能正在排阵推子，免罚
		r, c := g.PosToCoord(pos)
		skip := false
		for _, d := range board.ACTIONS {
			rr, cc := r+d[0], c+d[1]
			q := coordToPosSafe(g, rr, cc)
			if q >= 0 && g.TokenAt(q) == p {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		// 判断是否直接邻接 VOID
		for _, d := range board.ACTIONS {
			rr, cc := r+d[0], c+d[1]
			q := coordToPosSafe(g, rr, cc)
			if q >= 0 && g.TokenAt(q) == board.TokenVoid {
				bad += edgePenaltyStrong
				break
			}
		}
	}
	return bad
}

// Evaluate 计算 player 视角分数
// Evaluate 计算 player 视角分数（正分 = 有利）
func Evaluate(g *board.Game, player int8) int32 {

	opp := player ^ 1

	// ---------- hCapture：已捕获子差 ----------
	myPieces := float64(g.PlayerPieces(player))
	oppPieces := float64(g.PlayerPieces(opp))
	hCapture := capturedBonus * (myPieces - oppPieces) // 我方已多吃子 → 正分

	// ---------- h₁（中心距离） & hEdge ----------
	var dist [2]float64
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
	}
	h1 := (dist[opp] - dist[player]) * centerFactor

	// ---------- h₂（连通块差） ----------
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

	// ---------- h₃（子数差：备用） ----------
	h3 := myPieces - oppPieces

	// ---------- hPush ----------
	hPush := potentialPush(g, player) - potentialPush(g, opp)

	// ---------- 额外惩罚：贴边危险 ----------
	badSelf := edgePenalty(g, player)
	badOpp := edgePenalty(g, opp)

	// ---------- 综合 ----------
	absH1 := math.Abs(h1 / centerFactor)
	switch {
	case absH1 > switchHigh:
		return int32(h1 + h2 + hPush + hCapture + badOpp - badSelf)

	case absH1 < switchLow:
		return int32(h1 + hPush + hCapture + h3*materialHigh +
			badOpp - badSelf)

	default:
		return int32(h1 + h2 + hPush + hCapture + h3*materialMid +
			badOpp - badSelf)
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
// potentialPush 给出所有 Sumito 型阵列奖励（2vs1 / 3vs1 / 3vs2）
func potentialPush(g *board.Game, p int8) float64 {
	bonus := 0.0
	for pos := int8(0); pos < board.N; pos++ {
		if g.TokenAt(pos) != p {
			continue
		}
		r, c := g.PosToCoord(pos)
		for _, d := range board.ACTIONS {
			// 先取同向 3 格
			a1 := coordToPosSafe(g, r+d[0], c+d[1])
			a2 := coordToPosSafe(g, r+2*d[0], c+2*d[1])
			if a1 < 0 { // 至少得有 2 连
				continue
			}
			// 计算己子串长 (1/2/3)
			lenFriend := 1
			if g.TokenAt(a1) == p {
				lenFriend++
				if a2 >= 0 && g.TokenAt(a2) == p {
					lenFriend++
				}
			}
			// 敌方首格位置
			e1 := coordToPosSafe(g, r+int8(lenFriend)*d[0], c+int8(lenFriend)*d[1])
			if e1 < 0 || g.TokenAt(e1) != (p^1) {
				continue
			}
			// 第二个敌子（仅 3vs2 用）
			e2 := coordToPosSafe(g, r+int8(lenFriend+1)*d[0], c+int8(lenFriend+1)*d[1])
			lenEnemy := 1
			if lenFriend == 3 && e2 >= 0 && g.TokenAt(e2) == (p^1) {
				lenEnemy = 2
			}
			// 末尾必须为空格 / VOID / 越界 (表示可推进)
			tail := coordToPosSafe(g, r+int8(lenFriend+lenEnemy)*d[0], c+int8(lenFriend+lenEnemy)*d[1])
			if tail >= 0 && g.TokenAt(tail) != board.TokenEmpty && g.TokenAt(tail) != board.TokenVoid {
				continue
			}
			// Sumito 合法
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
