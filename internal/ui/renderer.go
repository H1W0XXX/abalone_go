// internal/ui/renderer.go
package ui

import (
	"abalone_go/internal/board"
	"bytes"
	_ "embed"
	"encoding/json"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// ────────────────────── 嵌入 PNG & JSON ──────────────────────

//go:embed assets/board.png
var boardPNG []byte

//go:embed assets/marbleA.png
var marbleAPNG []byte

//go:embed assets/marbleB.png
var marbleBPNG []byte

//go:embed assets/arrowA.png
var arrowAPNG []byte

//go:embed assets/arrowB.png
var arrowBPNG []byte

//go:embed assets/selected.png
var selectedPNG []byte

//go:embed assets/themes.json
var themesJSON []byte

// ────────────────────── 运行期缓存 ──────────────────────
var (
	boardImg, marbleAImg, marbleBImg  *ebiten.Image
	arrowAImg, arrowBImg, selectedImg *ebiten.Image

	cellCenters        [61][2]int
	outCoords          [2][6][2]int
	outCounts          [2]int
	selHalfW, selHalfH int
)

// ────────────────────── renderer ──────────────────────
type renderer struct{}

func newRenderer() *renderer {
	boardImg = imgFromBytes(boardPNG)
	marbleAImg = imgFromBytes(marbleAPNG)
	marbleBImg = imgFromBytes(marbleBPNG)
	arrowAImg = imgFromBytes(arrowAPNG)
	arrowBImg = imgFromBytes(arrowBPNG)
	selectedImg = imgFromBytes(selectedPNG)
	selHalfW = selectedImg.Bounds().Dx() / 2
	selHalfH = selectedImg.Bounds().Dy() / 2

	parseThemeJSON()
	return &renderer{}
}

func imgFromBytes(b []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	return ebiten.NewImageFromImage(img)
}

func parseThemeJSON() {
	var root map[string]any
	if err := json.Unmarshal(themesJSON, &root); err != nil {
		log.Fatal(err)
	}
	def := root["default"].(map[string]any)

	for i, v := range def["coordinates"].([]any) {
		xy := v.([]any)
		cellCenters[i][0] = int(xy[0].(float64))
		cellCenters[i][1] = int(xy[1].(float64))
	}
	outs := def["out_coordinates"].([]any)
	for p, arr := range outs {
		for k, v := range arr.([]any) {
			xy := v.([]any)
			outCoords[p][k][0] = int(xy[0].(float64))
			outCoords[p][k][1] = int(xy[1].(float64))
		}
	}
}

// drawBoard
func (r *renderer) drawBoard(screen *ebiten.Image, gl *GameLoop) {
	// 1) 背景棋盘
	screen.DrawImage(boardImg, nil)

	// 2) 静止棋子（跳过动画中的起点和终点）
	moving := make(map[int8]struct{}, len(gl.animating)*2)
	for _, anim := range gl.animating {
		moving[anim.from] = struct{}{}
		// anim.to<0 的话表示推出棋盘，不在 0~60 范围内
		if anim.to >= 0 {
			moving[anim.to] = struct{}{}
		}
	}
	for pos := int8(0); pos < 61; pos++ {
		if _, busy := moving[pos]; busy {
			continue
		}
		token := gl.logic.TokenAt(pos)
		if token == board.TokenEmpty || token == board.TokenVoid {
			continue
		}
		x := float64(cellCenters[int(pos)][0] - 24)
		y := float64(cellCenters[int(pos)][1] - 24)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		if token == board.PlayerA {
			screen.DrawImage(marbleAImg, op)
		} else {
			screen.DrawImage(marbleBImg, op)
		}
	}

	// 3) 选中高亮
	if sel := gl.input.selPos; sel >= 0 {
		px := float64(cellCenters[int(sel)][0] - selHalfW)
		py := float64(cellCenters[int(sel)][1] - selHalfH)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(px, py)
		screen.DrawImage(selectedImg, op)
	}

	// 4) 被推出的棋子 （保持不变）
	for p := 0; p < 2; p++ {
		for k := 0; k < outCounts[p]; k++ {
			ox := float64(outCoords[p][k][0] - 24)
			oy := float64(outCoords[p][k][1] - 24)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(ox, oy)
			if p == int(board.PlayerA) {
				screen.DrawImage(marbleAImg, op)
			} else {
				screen.DrawImage(marbleBImg, op)
			}
		}
	}

	// 5) 动画棋子（覆盖最上层）
	for _, a := range gl.animating {
		startXY := func() (float64, float64) {
			c := cellCenters[a.from]
			return float64(c[0] - 24), float64(c[1] - 24)
		}
		endXY := func() (float64, float64) {
			if a.to >= 0 { // 普通落子
				c := cellCenters[int(a.to)]
				return float64(c[0] - 24), float64(c[1] - 24)
			}
			// 推子：用 slotIdx 做索引
			idx := a.slotIdx // 0‥5
			oc := outCoords[int(a.piece)][idx]
			return float64(oc[0] - 24), float64(oc[1] - 24)
		}

		x, y, _ := a.screenXY(startXY, endXY)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		if a.piece == board.PlayerA {
			screen.DrawImage(marbleAImg, op)
		} else {
			screen.DrawImage(marbleBImg, op)
		}
	}
}

// applyModifications 需要 g 引用以确定玩家
func (r *renderer) applyModifications(mods []board.Modification, g *board.Game) {
	for _, m := range mods {
		if m.DirIndex == -1 {
			player := g.TokenAt(m.OldPos)
			if player == board.PlayerA || player == board.PlayerB {
				if outCounts[player] < 6 {
					outCounts[player]++
				}
			}
		}
	}
}
