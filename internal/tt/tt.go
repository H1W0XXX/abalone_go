// File: internal/tt/tt.go
package tt

import "math"

// Flag 表示评分类型：Exact(0)、LowerBound(1)、UpperBound(2)
type Flag uint8

const (
	Exact Flag = iota
	Lower
	Upper
)

// Entry 是 TT 中的一条记录
type Entry struct {
	Hash     uint64 // zobrist 哈希；用于碰撞校验
	Depth    int8   // 搜索深度
	Score    int32  // 节点评分
	Flag     Flag   // 上界/下界/精确
	BestMove uint32 // 紧凑表示的最佳着法（可自行定义）
}

// ———————————————————————————— 配置 ————————————————————————————

// 默认大小：2¹⁴ = 16,384 槽（约 16 k * 24 B ≈ 384 KB）
const defaultPow = 22

var (
	table     []Entry
	sizeMask  uint64
	emptyHash uint64 = 0 // Hash=0 视为“槽空”，因此 zobrist.Keys 绝不能生成 0
)

func init() { Resize(defaultPow) }

// Resize 重新创建 TT，容量 = 2^pow 个槽
// pow 越大占内存越多；pow=23 时占 ~8 Mi。
func Resize(pow uint8) {
	size := 1 << pow
	table = make([]Entry, size)
	sizeMask = uint64(size - 1)
}

// Clear 把所有槽标记为空
func Clear() {
	for i := range table {
		table[i].Hash = emptyHash
	}
}

// ———————————————————————————— API ————————————————————————————

// Probe 查表：
//
//	hit == false → 不命中
//	若 flag==Exact, score 即为精确值
//	若 flag==Lower/Upper，可配合 αβ 取代 alpha/beta
func Probe(hash uint64, depth int8, alpha, beta int32) (hit bool, score int32, flag Flag, best uint32) {
	e := &table[hash&sizeMask]
	if e.Hash == hash {
		// 同一局面
		if e.Depth >= depth { // 只在条目深度 ≥ 需求深度时才可直接用
			return true, e.Score, e.Flag, e.BestMove
		}
	}
	return false, 0, Exact, 0
}

// Store 把新搜索结果写入表
func Store(hash uint64, depth int8, score int32, flag Flag, best uint32) {
	e := &table[hash&sizeMask]
	// 替换策略：若槽空或深度更大，就覆盖
	if e.Hash == emptyHash || depth >= e.Depth {
		e.Hash, e.Depth, e.Score, e.Flag, e.BestMove = hash, depth, score, flag, best
	}
}

// MateScore / UnmateScore 辅助
// 用于把搜索返回的“将杀距离分数”规范化（αβ常见技巧）
const (
	mateValue  = math.MaxInt32 / 2 // 代表“极大”分
	mateBuffer = 10000             // 距离步数编码偏移
)

// ToTTScore 把 engine 内部的 ±mate 分数编码进 TT
func ToTTScore(score int32, plyFromRoot int32) int32 {
	if score > mateValue-mateBuffer {
		// 正 mate
		return score + plyFromRoot
	}
	if score < -mateValue+mateBuffer {
		// 负 mate
		return score - plyFromRoot
	}
	return score
}

// FromTTScore 把 TT 里的特殊分数还原回 engine 评分
func FromTTScore(score int32, plyFromRoot int32) int32 {
	if score > mateValue-mateBuffer {
		return score - plyFromRoot
	}
	if score < -mateValue+mateBuffer {
		return score + plyFromRoot
	}
	return score
}
