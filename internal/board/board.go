// File internal/board/board.go
package board

const (
	N          = 61 // 可落子格子数
	TokenVoid  = int8(-2)
	TokenEmpty = int8(-1)
	PlayerA    = int8(0)
	PlayerB    = int8(1)

	boardSize = 11
	lifes     = 6
)

// ACTIONS 六个方向 (dr, dc)
var ACTIONS = [6][2]int8{
	{0, 1},  // RIGHT
	{1, 0},  // DOWN_RIGHT
	{1, -1}, // DOWN_LEFT
	{0, -1}, // LEFT
	{-1, 0}, // UP_LEFT
	{-1, 1}, // UP_RIGHT
}

type Game struct {
	Cells           [boardSize][boardSize]int8 // 与 python 同形的 11×11 矩阵
	posIndex        [N][2]int8                 // index -> (r,c)
	coordIndex      [boardSize][boardSize]int8 // (r,c) -> index / -1
	playerDamages   [2]int8
	CurrentPlayer   int8
	TurnCount       int
	GameOver        bool
	PlayerVictories [2]int
}

// --------------------- 构造 & 初始化 ------------------------

func NewGame(startPlayer int8) *Game {
	g := &Game{}
	g.initCoordTables()
	g.reset(startPlayer)
	return g
}

func (g *Game) initCoordTables() {
	for r := range g.coordIndex {
		for c := range g.coordIndex[r] {
			g.coordIndex[r][c] = -1
		}
	}

	// 找空格位置，与 python 的 positions 顺序保持一致
	idx := 0
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			g.Cells[r][c] = TokenVoid // 默认都设 VOID
		}
	}
	// 内部可下子区域：与 Python 的 new_board() 完全一致
	for r := 1; r < boardSize-1; r++ {
		for c := 1; c < boardSize-1; c++ {
			// ① 先做大菱形裁剪
			if r+c < 5 || r+c > 15 {
				continue
			}
			// ② 再扣掉左上 4 层三角
			if r <= 4 && c <= 5-r {
				continue
			}
			// ③ 再扣掉右下 4 层三角
			//    行 6→c≥9, 行 7→c≥8, 行 8→c≥7, 行 9→c≥6
			if r >= 6 && c >= 15-r {
				continue
			}

			g.coordIndex[r][c] = int8(idx)
			g.posIndex[idx] = [2]int8{int8(r), int8(c)}
			g.Cells[r][c] = TokenEmpty
			idx++
		}
	}

}

func (g *Game) reset(startPlayer int8) {
	// 清空棋盘
	for r := range g.Cells {
		for c := range g.Cells[r] {
			if g.Cells[r][c] != TokenVoid {
				g.Cells[r][c] = TokenEmpty
			}
		}
	}
	// Classical 方案：黑白双方各 14 子
	initialA := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	initialB := []int{47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60}
	for _, p := range initialA {
		r, c := g.posIndex[p][0], g.posIndex[p][1]
		g.Cells[r][c] = PlayerA
	}
	for _, p := range initialB {
		r, c := g.posIndex[p][0], g.posIndex[p][1]
		g.Cells[r][c] = PlayerB
	}

	g.playerDamages = [2]int8{}
	g.CurrentPlayer = startPlayer
	g.TurnCount = 1
	g.GameOver = false
}

// -------------------- 公共工具 -----------------------------

// PosToCoord 把索引映射到 (r,c)
func (g *Game) PosToCoord(pos int8) (int8, int8) { rc := g.posIndex[pos]; return rc[0], rc[1] }

// CoordToPos 若坐标非法返回 -1
func (g *Game) CoordToPos(r, c int8) int8 { return g.coordIndex[r][c] }

// TokenAt 直接读棋子
func (g *Game) TokenAt(pos int8) int8 {
	r, c := g.PosToCoord(pos)
	return g.Cells[r][c]
}
