// File internal/board/rules.go
package board

import "math"

// ---------------- 内部辅助 -----------------

// inlineDecompose 与 python 版 decompose_inline 相同
func inlineDecompose(dr, dc int8) (step, dir int8, ok bool) {
	switch {
	case dr == 0 && dc != 0: // ↔
		return abs(dc), tern(dc > 0, int8(0), int8(3)), true
	case dc == 0 && dr != 0: // ↕ 斜向
		return abs(dr), tern(dr > 0, int8(1), int8(4)), true
	case abs(dr) == abs(dc) && dr != 0 && dc != 0 && sign(dr) != sign(dc): // ↘↖
		return abs(dr), tern(dr > 0, int8(2), int8(5)), true
	default:
		return 0, 0, false
	}
}

func sign(x int8) int8 {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}
func abs(x int8) int8 {
	if x < 0 {
		return -x
	}
	return x
}
func tern(cond bool, a, b int8) int8 {
	if cond {
		return a
	}
	return b
}

// --------------- Move 验证主入口 ----------------

// ValidateMove 与 Python 的返回语义保持一致：
// ok=false 不合法；否则返回 moveType 和 modifications slice。
type Modification struct {
	OldPos, NewPos int8 // NewPos=-1 表示被推出棋盘
	DirIndex       int8 // 0-5 方向；-1 代表 eject
}

func (g *Game) ValidateMove(pos0, pos1 int8) (ok bool, moveType string, mods []Modification) {
	player := g.CurrentPlayer
	r0, c0 := g.PosToCoord(pos0)
	r1, c1 := g.PosToCoord(pos1)
	if g.Cells[r0][c0] != player || g.Cells[r1][c1] == player {
		return
	}

	switch g.Cells[r1][c1] {
	case TokenEmpty:
		if mt, m := g.inlineMove(r0, c0, r1, c1); mt != "" {
			return true, mt, m
		}
		if mt, m := g.sideStepMove(r0, c0, r1, c1); mt != "" {
			return true, mt, m
		}
	default: // 敌子
		if mt, m := g.inlinePush(r0, c0, r1, c1); mt != "" {
			return true, mt, m
		}
	}
	return
}

// ------------- 三种走法检查 ------------------

func (g *Game) inlineMove(r0, c0, r1, c1 int8) (string, []Modification) {
	// Δ 坐标
	dr, dc := r1-r0, c1-c0
	step, dir, ok := inlineDecompose(dr, dc)
	if !ok || step == 0 || step > 3 {
		return "", nil
	}
	rStep, cStep := ACTIONS[dir][0], ACTIONS[dir][1]

	// 1) 检查整条线都是当前玩家
	for n := int8(0); n < step; n++ {
		rr, cc := r0+n*rStep, c0+n*cStep
		if g.Cells[rr][cc] != g.CurrentPlayer {
			return "", nil
		}
	}

	// 2) oldPos：从尾到头（反序）
	oldPos := make([]int8, 0, step)
	for n := step - 1; n >= 0; n-- {
		rr, cc := r0+n*rStep, c0+n*cStep
		oldPos = append(oldPos, g.CoordToPos(rr, cc))
	}

	// 3) newPos：目标格 + 其余依次向前
	destPos := g.CoordToPos(r1, c1)
	newPos := append([]int8{destPos}, oldPos[:len(oldPos)-1]...)

	// 4) 构造 modifications
	mods := make([]Modification, len(oldPos))
	for i := range oldPos {
		mods[i] = Modification{
			OldPos:   oldPos[i],
			NewPos:   newPos[i],
			DirIndex: dir,
		}
	}
	return "inline_move", mods
}

// inlinePush 实现了 Python 版的 “Sumito” 规则：2vs1、3vs1、3vs2 推子逻辑。
func (g *Game) inlinePush(r0, c0, r1, c1 int8) (string, []Modification) {
	dr, dc := r1-r0, c1-c0
	step, dir, ok := inlineDecompose(dr, dc)
	if !ok || step > 4 {
		return "", nil
	}
	rStep, cStep := ACTIONS[dir][0], ACTIONS[dir][1]

	//---------------- 1) 收集连续棋串 ------------------
	tail := [][]int8{{r0, c0}} // tail[0] 我方，tail[1] 敌方
	prev := g.CurrentPlayer
	reached := false
	rr, cc := r0+rStep, c0+cStep
	for {
		if g.Cells[rr][cc] == TokenEmpty || g.Cells[rr][cc] == TokenVoid {
			break
		}
		if g.Cells[rr][cc] != prev {
			tail = append(tail, []int8{rr, cc})
			prev = g.Cells[rr][cc]
		} else {
			last := len(tail) - 1
			tail[last] = append(tail[last], rr, cc)
		}
		if rr == r1 && cc == c1 {
			reached = true
		}
		rr += rStep
		cc += cStep
	}

	//---------------- 2) 合规校验 -----------------------
	if !reached || len(tail) != 2 {
		return "", nil
	}
	nFriends, nEnemies := len(tail[0])/2, len(tail[1])/2
	if nFriends <= nEnemies || nFriends > 3 {
		return "", nil
	}

	//---------------- 3) 处理顶出棋 ---------------------
	mods := []Modification{}
	destR, destC := rr, cc // 末尾的空格 / VOID
	moveType := "inline_push"
	if g.Cells[rr][cc] == TokenVoid { // 有顶出
		outR, outC := rr-rStep, cc-cStep // 最后那颗敌棋
		outPos := g.CoordToPos(outR, outC)
		mods = append(mods, Modification{ // 先记录 ejection
			OldPos: outPos, NewPos: -1, DirIndex: -1,
		})
		// 更新胜负标记
		damagedPlayer := g.Cells[outR][outC]
		if g.playerDamages[damagedPlayer]+1 == lifes {
			moveType = "winner"
		} else {
			moveType = "ejected"
		}
		destR, destC = outR, outC // 链式移动的“空格”换成刚空出来的位置
		// 把被顶出的坐标从“敌方串”里剪掉
		tail[1] = tail[1][:len(tail[1])-2]
	}

	//---------------- 4) 构造移动链 ----------------------
	// 把 (友方串 + 敌方串) 拉平成一维，再整体反转
	chain := append([]int8{}, tail[0]...) // friend coords
	chain = append(chain, tail[1]...)     // enemy coords
	for i, j := 0, len(chain)-2; i < j; i, j = i+2, j-2 {
		chain[i], chain[j] = chain[j], chain[i]
		chain[i+1], chain[j+1] = chain[j+1], chain[i+1]
	}

	// 目标格（空格）先转索引
	destPos := g.CoordToPos(destR, destC)

	// oldPos / newPos 一一对应
	oldPos := make([]int8, len(chain)/2)
	for i := 0; i < len(chain); i += 2 {
		oldPos[i/2] = g.CoordToPos(chain[i], chain[i+1])
	}
	newPos := append([]int8{destPos}, oldPos[:len(oldPos)-1]...)

	for i := range oldPos {
		mods = append(mods, Modification{
			OldPos: oldPos[i], NewPos: newPos[i], DirIndex: dir,
		})
	}

	return moveType, mods
}

// ----------------- 盘面修改 ---------------------

func (g *Game) Apply(mods []Modification) {
	if len(mods) == 0 {
		return
	}
	for _, m := range mods {
		if m.NewPos == -1 { // eject
			r, c := g.PosToCoord(m.OldPos)
			damaged := g.Cells[r][c]
			g.playerDamages[damaged]++
			g.Cells[r][c] = TokenEmpty
			if g.playerDamages[damaged] == lifes {
				g.GameOver = true
				g.PlayerVictories[g.CurrentPlayer]++
			}
			continue
		}
		rOld, cOld := g.PosToCoord(m.OldPos)
		rNew, cNew := g.PosToCoord(m.NewPos)
		g.Cells[rNew][cNew], g.Cells[rOld][cOld] = g.Cells[rOld][cOld], TokenEmpty
	}
	g.CurrentPlayer ^= 1
	g.TurnCount++
}

// 帮助把 [][2]int8 坐标切片转成 pos 切片
func (g *Game) CoordToPosSlice(coords [][2]int8) []int8 {
	out := make([]int8, 0, len(coords))
	for _, p := range coords {
		out = append(out, g.CoordToPos(p[0], p[1]))
	}
	return out
}

// sideStepMove 对应 Python 版的 check_sidestep_move，检查并返回“侧移”走法。
// 如果合法，返回 moveType="sidestep_move" 和 modifications 列表；否则返回 "", nil。
func (g *Game) sideStepMove(r0, c0, r1, c1 int8) (string, []Modification) {
	// dr,dc 为目标相对起点的偏移
	dr, dc := r1-r0, c1-c0

	// 1. 预先计算 tmp[i] = inlineDecompose(dr - dr_i, dc - dc_i)
	//    tmp[i][0] 存 step，tmp[i][1] 存 dir。忽略 ok 标记，默认 ok=false 时 step=0, dir=0。
	var tmp [6][2]int8
	for i, d := range ACTIONS {
		s, dir, _ := inlineDecompose(dr-d[0], dc-d[1])
		tmp[i][0], tmp[i][1] = s, dir
	}

	// 2. 计算三轴方向分解
	d0 := g.decomposeDirections(r0, c0) // [3]int8
	d1 := g.decomposeDirections(r1, c1)
	var decomp [3]int8
	for i := 0; i < 3; i++ {
		decomp[i] = d1[i] - d0[i]
	}

	// 3. 依次尝试三个轴，找到 abs(decomp[i]) == 1 的情况
	//    并根据正负方向选取对应的两个候选 side_move 指向
	actP := [3][2]int8{{1, 2}, {0, 5}, {0, 1}}
	actN := [3][2]int8{{4, 5}, {3, 2}, {3, 4}}

	for i := 0; i < 3; i++ {
		if absInt8(decomp[i]) != 1 {
			continue
		}
		// 根据正负选取候选
		var sm0, sm1 int8
		if decomp[i] > 0 {
			sm0, sm1 = actP[i][0], actP[i][1]
		} else {
			sm0, sm1 = actN[i][0], actN[i][1]
		}
		// 选择 side_move，使得 inline step 更小
		var sideMove int8
		if tmp[sm0][0] < tmp[sm1][0] {
			sideMove = sm0
		} else {
			sideMove = sm1
		}

		// 4. 计算 side_move 对应的移动向量
		drStep, dcStep := ACTIONS[sideMove][0], ACTIONS[sideMove][1]
		inlineStep, inlineMove := tmp[sideMove][0], tmp[sideMove][1]
		drInline, dcInline := ACTIONS[inlineMove][0], ACTIONS[inlineMove][1]

		// 5. 限制：inlineStep < 3
		if inlineStep >= 3 {
			continue
		}

		// 6. 构造 oldCoords 与 newCoords 列表
		//    oldCoords: [(r0 + k*drInline, c0 + k*dcInline) for k in 0..inlineStep]
		//    newCoords: [(r0 + drStep + k*drInline, c0 + dcStep + k*dcInline) for k in 0..inlineStep]
		oldCoords := make([][2]int8, inlineStep+1)
		newCoords := make([][2]int8, inlineStep+1)
		for k := int8(0); k <= inlineStep; k++ {
			oldCoords[k][0] = r0 + drInline*k
			oldCoords[k][1] = c0 + dcInline*k
			newCoords[k][0] = r0 + drStep + drInline*k
			newCoords[k][1] = c0 + dcStep + dcInline*k
		}

		// 7. 检查 connected：oldCoords 全部都是当前玩家的棋
		connected := true
		for _, rc := range oldCoords {
			if g.Cells[rc[0]][rc[1]] != g.CurrentPlayer {
				connected = false
				break
			}
		}
		if !connected {
			continue
		}

		// 8. 检查 free：newCoords 全部为空格
		free := true
		for _, rc := range newCoords {
			if g.Cells[rc[0]][rc[1]] != TokenEmpty {
				free = false
				break
			}
		}
		if !free {
			continue
		}

		// 9. 构造 modifications 列表
		var mods []Modification
		oldPositions := make([]int8, inlineStep+1)
		newPositions := make([]int8, inlineStep+1)
		for k := int8(0); k <= inlineStep; k++ {
			oldPositions[k] = g.CoordToPos(oldCoords[k][0], oldCoords[k][1])
			newPositions[k] = g.CoordToPos(newCoords[k][0], newCoords[k][1])
		}
		for idx := 0; idx < int(inlineStep+1); idx++ {
			mods = append(mods, Modification{
				OldPos:   oldPositions[idx],
				NewPos:   newPositions[idx],
				DirIndex: sideMove,
			})
		}

		// 10. 返回合法的 sidestep_move
		return "sidestep_move", mods
	}

	return "", nil
}

// decomposeDirections 对应 Python 版的 decompose_directions(r, c)
func (g *Game) decomposeDirections(r, c int8) [3]int8 {
	return [3]int8{r, c, r + c - 4}
}

// absInt8 取绝对值
func absInt8(x int8) int8 {
	return int8(math.Abs(float64(x)))
}
