// File: internal/search/search.go
package search

import (
	"math"
	"runtime"
	"sync"
	"time"

	"abalone_go/internal/board"
	"abalone_go/internal/eval"
	"abalone_go/internal/tt"
	"abalone_go/internal/zobrist"
)

const mateValue = 32000 // 必须与 tt.ToTTScore 内部阈值保持一致

/* ───────────────────── 公开 API ────────────────────── */

// BestMove（单线程）保留不变 —— 给低端 CPU 或调试用
func BestMove(root *board.Game, depth int8, limit time.Duration) (best0, best1 int8, bestScore int32, found bool) {
	return bestCore(root, depth, limit, 1)
}

// BestMoveParallel：根层并行，worker=CPU-1
func BestMoveParallel(root *board.Game, depth int8, limit time.Duration) (int8, int8, int32, bool) {
	workers := runtime.NumCPU() - 1
	if workers < 1 {
		workers = 1
	}
	return bestCore(root, depth, limit, workers)
}

/* ─────────────────── 内部并行核心 ──────────────────── */

func bestCore(root *board.Game, depth int8, limit time.Duration, workers int) (int8, int8, int32, bool) {
	//start := time.Now()
	runtime.GOMAXPROCS(workers + 1) // +1 给 UI/main 线程

	// 先把根节点所有可行走法搜出来
	type mv struct{ from, to int8 }
	var moves []mv
	for from := int8(0); from < board.N; from++ {
		for to := int8(0); to < board.N; to++ {
			if ok, _, _ := root.ValidateMove(from, to); ok {
				moves = append(moves, mv{from, to})
			}
		}
	}
	if len(moves) == 0 {
		return -1, -1, 0, false
	}

	// 通道派工
	taskCh := make(chan mv, len(moves))
	resCh := make(chan result, len(moves))

	for w := 0; w < workers; w++ {
		go func() {
			for mv := range taskCh {
				child := *root
				ok, _, mods := child.ValidateMove(mv.from, mv.to)
				if !ok {
					continue
				}
				child.Apply(mods)

				h := zobrist.HashFromCells(flatCells(&child))
				score, _, _ := alphabeta(&child, h, depth-1, math.MinInt32, math.MaxInt32, 1)
				score = -score // 反视角

				resCh <- result{score: score, from: mv.from, to: mv.to}
			}
		}()
	}

	// 投递任务
	for _, m := range moves {
		taskCh <- m
	}
	close(taskCh)

	// 聚合结果
	best := result{score: math.MinInt32}
	timeout := time.After(limit)

	for done := 0; done < len(moves); done++ {
		select {
		case r := <-resCh:
			if r.score > best.score {
				best = r
			}
			if abs32(r.score) > mateValue-500 { // 找杀直接提前结束
				return best.from, best.to, best.score, true
			}
		case <-timeout:
			return best.from, best.to, best.score, true // 返回当前最优
		}
	}
	return best.from, best.to, best.score, true
}

/* ──────────────────── αβ & TT (线程安全包装) ──────────────────── */

var ttMu sync.RWMutex // 读多写少，用 RWMutex 更合适

func alphabeta(node *board.Game, hash uint64, depth int8, alpha, beta int32, ply int8) (int32, int8, int8) {
	alphaOrig := alpha

	if depth == 0 || node.GameOver {
		return eval.Evaluate(node, node.CurrentPlayer), -1, -1
	}

	/* ------- TT probe (加读锁) ------- */
	var (
		hit      bool
		ttVal    int32
		flag     tt.Flag
		bestMove uint32
	)
	ttMu.RLock()
	hit, ttVal, flag, bestMove = tt.Probe(hash, depth, alpha, beta)
	ttMu.RUnlock()
	if hit {
		score := tt.FromTTScore(ttVal, int32(ply))
		switch flag {
		case tt.Exact:
			return score, int8(bestMove >> 8), int8(bestMove & 0xFF)
		case tt.Lower:
			alpha = max32(alpha, score)
		case tt.Upper:
			beta = min32(beta, score)
		}
		if alpha >= beta {
			return score, int8(bestMove >> 8), int8(bestMove & 0xFF)
		}
	}

	/* ------- 生成走法（暴力） ------- */
	bestScore := int32(math.MinInt32)
	var bestFrom, bestTo int8 = -1, -1

outer:
	for from := int8(0); from < board.N; from++ {
		for to := int8(0); to < board.N; to++ {
			ok, _, mods := node.ValidateMove(from, to)
			if !ok {
				continue
			}
			child := *node
			child.Apply(mods)
			newHash := zobrist.HashFromCells(flatCells(&child))

			score, _, _ := alphabeta(&child, newHash, depth-1, -beta, -alpha, ply+1)
			score = -score
			if score > bestScore {
				bestScore, bestFrom, bestTo = score, from, to
			}
			alpha = max32(alpha, score)
			if alpha >= beta {
				break outer
			}
		}
	}

	/* ------- 写 TT (加写锁) ------- */
	flagTT := tt.Exact
	if bestScore <= alphaOrig {
		flagTT = tt.Upper
	} else if bestScore >= beta {
		flagTT = tt.Lower
	}
	enc := tt.ToTTScore(bestScore, int32(ply))
	ttMu.Lock()
	tt.Store(hash, depth, enc, flagTT, uint32(bestFrom)<<8|uint32(bestTo))
	ttMu.Unlock()

	return bestScore, bestFrom, bestTo
}

/* ─────────────────── 小工具 ─────────────────── */

func flatCells(g *board.Game) []int8 {
	flat := make([]int8, 0, board.N)
	for pos := int8(0); pos < board.N; pos++ {
		flat = append(flat, g.TokenAt(pos))
	}
	return flat
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
func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
func abs32(a int32) int32 {
	if a < 0 {
		return -a
	}
	return a
}
