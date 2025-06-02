// File: internal/zobrist/zobrist.go
package zobrist

import (
	"math/rand"
	"time"
)

const (
	Players   = 2  // A / B
	Positions = 61 // 可落子格子数（固定）
)

var Keys [Players][Positions]uint64

func init() {
	// 用 time.Now 纳秒做种子；如需确定性测试，以固定常量替换即可
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for p := 0; p < Players; p++ {
		for i := 0; i < Positions; i++ {
			// 避免生成 0（XOR 不起作用）
			v := rng.Uint64()
			for v == 0 {
				v = rng.Uint64()
			}
			Keys[p][i] = v
		}
	}
}

// Toggle 对 (player, pos) 的键做一次 XOR，并返回新哈希。
// 用法：hash = zobrist.Toggle(hash, player, oldPos)  // 移走
//
//	hash = zobrist.Toggle(hash, player, newPos)  // 落下
func Toggle(hash uint64, player, pos int8) uint64 {
	return hash ^ Keys[player][pos]
}

// HashFromCells 根据一维 cells 切片计算整盘哈希。
// cells[pos] == 0 / 1 表示棋子；其它值（空 / VOID）会被忽略。
func HashFromCells(cells []int8) uint64 {
	var h uint64
	for pos, token := range cells {
		if token == 0 || token == 1 { // 只对玩家 A/B 做 XOR
			h ^= Keys[token][pos]
		}
	}
	return h
}
