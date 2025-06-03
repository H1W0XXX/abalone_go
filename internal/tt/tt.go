package tt

import "math"

/* ————————— 条目 ————————— */

type Flag uint8

const (
	Exact Flag = iota
	Lower
	Upper
)

type Entry struct {
	Hash     uint64 // 必须排在最前，保证 8 字节对齐
	Depth    int8
	Score    int32
	Flag     Flag
	BestMove uint32
}

/* ————————— 参数 ————————— */

const defaultPow = 22 // 4 MiB: 2^22 × 24 B ≈ 100 Mi

var (
	table     []Entry
	sizeMask  uint64
	emptyHash uint64 = 0
)

func init() { Resize(defaultPow) }

func Resize(pow uint8) {
	n := 1 << pow
	table = make([]Entry, n)
	sizeMask = uint64(n - 1)
}

func Clear() {
	for i := range table {
		table[i].Hash = emptyHash
	}
}

/* ————————— 无锁 API ————————— */

// Probe：无锁读；读到脏数据会被 Hash 校验挡下
func Probe(hash uint64, depth int8, alpha, beta int32) (bool, int32, Flag, uint32) {
	e := &table[hash&sizeMask]

	if e.Hash == hash && e.Depth >= depth {
		return true, e.Score, e.Flag, e.BestMove
	}
	return false, 0, Exact, 0
}

// Store：无锁写；直接覆盖
func Store(hash uint64, depth int8, score int32, flag Flag, best uint32) {
	e := &table[hash&sizeMask]

	// 避免 Hash 为 0（视为空）
	if hash == emptyHash {
		hash = 1
	}

	// 简单替换策略：空槽或更深就覆盖
	if e.Hash == emptyHash || depth >= e.Depth {
		e.Hash = hash // ① 先写 Hash
		e.Score = score
		e.Flag = flag
		e.BestMove = best
		e.Depth = depth // ② 最后写 Depth（读侧先看 Depth）
	}
}

/* ————————— Mate ↔ Score ————————— */

const (
	mateValue  = math.MaxInt32 / 2
	mateBuffer = 10000
)

func ToTTScore(s, ply int32) int32 {
	if s > mateValue-mateBuffer {
		return s + ply
	}
	if s < -mateValue+mateBuffer {
		return s - ply
	}
	return s
}
func FromTTScore(s, ply int32) int32 {
	if s > mateValue-mateBuffer {
		return s - ply
	}
	if s < -mateValue+mateBuffer {
		return s + ply
	}
	return s
}
