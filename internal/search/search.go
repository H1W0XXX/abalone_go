// internal/search/search.go
package search

import (
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"abalone_go/internal/board"
	"abalone_go/internal/eval"
	"abalone_go/internal/tt"
	"abalone_go/internal/zobrist"
)

const mateValue = 32000

// —— 共用走法结构 ——
type mv struct{ from, to int8 }

/* ──────────────── 公开 API ──────────────── */

func BestMove(root *board.Game, depth int8, limit time.Duration) (int8, int8, int32, bool) {
	return bestCore(root, depth, limit, 1)
}
func BestMoveParallel(root *board.Game, depth int8, limit time.Duration) (int8, int8, int32, bool) {
	w := runtime.NumCPU() - 1
	if w < 1 {
		w = 1
	}
	return bestCore(root, depth, limit, w)
}

/* ──────────────── 并行根层 ──────────────── */

func bestCore(root *board.Game, depth int8, limit time.Duration, workers int) (int8, int8, int32, bool) {
	runtime.GOMAXPROCS(workers + 1)

	moves := orderMoves(root, genMoves(root))
	if len(moves) == 0 {
		return -1, -1, 0, false
	}

	taskCh := make(chan mv, len(moves))
	resCh := make(chan result, len(moves))

	for w := 0; w < workers; w++ {
		go func() {
			for m := range taskCh {
				child := *root
				if ok, _, mods := child.ValidateMove(m.from, m.to); ok {
					child.Apply(mods)
					h := zobrist.HashFromCells(flatCells(&child))
					score, _ := pvs(&child, h, depth-1, -mateValue, mateValue, 1, false)
					resCh <- result{-score, m.from, m.to} // 反视角
				}
			}
		}()
	}
	for _, m := range moves {
		taskCh <- m
	}
	close(taskCh)

	best := result{score: math.MinInt32}
	timeout := time.After(limit)
	for done := 0; done < len(moves); done++ {
		select {
		case r := <-resCh:
			if r.score > best.score {
				best = r
			}
			if abs32(r.score) > mateValue-500 {
				return best.from, best.to, best.score, true
			}
		case <-timeout:
			return best.from, best.to, best.score, true
		}
	}
	return best.from, best.to, best.score, true
}

/* ──────────────── PVS + NM + LMR + QSearch ──────────────── */

var ttMu sync.RWMutex

func pvs(node *board.Game, hash uint64, depth int8, alpha, beta int32, ply int8, isPV bool) (int32, uint32) {
	/* --- Quiescence --- */
	if depth == 0 || node.GameOver {
		return quiesce(node, alpha, beta, ply), 0
	}

	/* --- Null-Move (禁止在 PV) --- */
	if !isPV && depth >= 3 {
		null := *node
		null.CurrentPlayer ^= 1 // 让一手
		score, _ := pvs(&null, hash^0xABCDEF, depth-3, -beta, -beta+1, ply+1, false)
		if -score >= beta {
			return beta, 0
		}
	}

	/* --- TT Probe --- */
	if s, mv, ok := ttProbe(hash, depth, alpha, beta, ply); ok {
		return s, mv
	}

	bestScore := int32(math.MinInt32)
	var bestMove uint32
	moveCount := 0

	for _, m := range orderMoves(node, genMoves(node)) {
		moveCount++
		child := *node
		ok, _, mods := child.ValidateMove(m.from, m.to)
		if !ok {
			continue
		}
		child.Apply(mods)
		newHash := zobrist.HashFromCells(flatCells(&child))

		/* --- LMR: 后继第4手起、非PV、深度≥3 减 1 --- */
		reduce := int8(0)
		if depth >= 3 && !isPV && moveCount > 3 {
			reduce = 1
		}

		var score int32
		if moveCount == 1 { // 首子用全窗
			score, _ = pvs(&child, newHash, depth-1, -beta, -alpha, ply+1, true)
			score = -score
		} else {
			// 先零窗
			score, _ = pvs(&child, newHash, depth-1-reduce, -alpha-1, -alpha, ply+1, false)
			score = -score
			if score > alpha && reduce > 0 { // LMR 提升
				score, _ = pvs(&child, newHash, depth-1, -alpha-1, -alpha, ply+1, false)
				score = -score
			}
			if score > alpha && score < beta { // 窄窗失败高，再全窗
				score, _ = pvs(&child, newHash, depth-1, -beta, -alpha, ply+1, true)
				score = -score
			}
		}

		if score > bestScore {
			bestScore, bestMove = score, uint32(m.from)<<8|uint32(m.to)
		}
		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			break // β 剪
		}
	}

	/* --- TT Store --- */
	ttStore(hash, depth, bestScore, alphaOrig(alpha), beta, bestMove, ply)

	return bestScore, bestMove
}

/* ----- Quiescence: 只扩展 inline_push ----- */
func quiesce(node *board.Game, alpha, beta int32, ply int8) int32 {
	stand := eval.Evaluate(node, node.CurrentPlayer)
	if stand >= beta {
		return beta
	}
	if stand > alpha {
		alpha = stand
	}

	for _, m := range genMoves(node) {
		if moveType(node, m.from, m.to) != "inline_push" {
			continue
		}
		child := *node
		ok, _, mods := child.ValidateMove(m.from, m.to)
		if !ok {
			continue
		}
		child.Apply(mods)
		score := -quiesce(&child, -beta, -alpha, ply+1)
		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}
	return alpha
}

/* ----- TT helpers ----- */

func ttProbe(hash uint64, depth int8, alpha, beta int32, ply int8) (int32, uint32, bool) {
	ttMu.RLock()
	hit, v, flag, mv := tt.Probe(hash, depth, alpha, beta)
	ttMu.RUnlock()
	if !hit {
		return 0, 0, false
	}
	score := tt.FromTTScore(v, int32(ply))
	switch flag {
	case tt.Exact:
		return score, mv, true
	case tt.Lower:
		if score > alpha {
			alpha = score
		}
	case tt.Upper:
		if score < beta {
			beta = score
		}
	}
	if alpha >= beta {
		return score, mv, true
	}
	return 0, 0, false
}

func ttStore(hash uint64, depth int8, score, alpha, beta int32, mv uint32, ply int8) {
	flag := tt.Exact
	if score <= alpha {
		flag = tt.Upper
	} else if score >= beta {
		flag = tt.Lower
	}
	val := tt.ToTTScore(score, int32(ply))
	ttMu.Lock()
	tt.Store(hash, depth, val, flag, mv)
	ttMu.Unlock()
}

func alphaOrig(a int32) int32 { return a } // 留做可读替身

/* ──────────────── 工具 & 排序 ──────────────── */

func genMoves(g *board.Game) []mv {
	out := make([]mv, 0, 128)
	for f := int8(0); f < board.N; f++ {
		for t := int8(0); t < board.N; t++ {
			if ok, _, _ := g.ValidateMove(f, t); ok {
				out = append(out, mv{f, t})
			}
		}
	}
	return out
}

func flatCells(g *board.Game) []int8 {
	arr := make([]int8, board.N)
	for i := int8(0); i < board.N; i++ {
		arr[i] = g.TokenAt(i)
	}
	return arr
}

type result struct {
	score    int32
	from, to int8
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
func abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

/* ---------- 排序 (推子>三连>侧移>普通) ---------- */

func moveType(g *board.Game, from, to int8) string {
	if ok, mt, _ := g.ValidateMove(from, to); ok {
		return mt
	}
	return ""
}

func makesLine3(g *board.Game, m mv) bool {
	tmp := *g
	if ok, _, mods := tmp.ValidateMove(m.from, m.to); ok {
		tmp.Apply(mods)
	} else {
		return false
	}
	for p := int8(0); p < board.N; p++ {
		if tmp.TokenAt(p) != g.CurrentPlayer {
			continue
		}
		r, c := tmp.PosToCoord(p)
		for dir, d := range board.ACTIONS[:3] { // 3 方向即可
			p1 := safePos(&tmp, r+d[0], c+d[1])
			p2 := safePos(&tmp, r+2*d[0], c+2*d[1])
			if p1 >= 0 && p2 >= 0 &&
				tmp.TokenAt(p1) == g.CurrentPlayer &&
				tmp.TokenAt(p2) == g.CurrentPlayer {
				return true
			}
			if dir >= 2 {
				break
			}
		}
	}
	return false
}
func safePos(g *board.Game, r, c int8) int8 {
	if r < 0 || r > 10 || c < 0 || c > 10 {
		return -1
	}
	return g.CoordToPos(r, c)
}

func orderMoves(g *board.Game, list []mv) []mv {
	type s struct {
		mv mv
		sc int
	}
	buf := make([]s, 0, len(list))
	for _, m := range list {
		score := 0
		switch moveType(g, m.from, m.to) {
		case "inline_push":
			score = 5000
		case "sidestep_move":
			score = 3000
		case "inline_move":
			score = 1000
		}
		if makesLine3(g, m) {
			score += 2000
		}
		buf = append(buf, s{m, score})
	}
	sort.Slice(buf, func(i, j int) bool { return buf[i].sc > buf[j].sc })
	out := make([]mv, len(buf))
	for i, v := range buf {
		out[i] = v.mv
	}
	return out
}
