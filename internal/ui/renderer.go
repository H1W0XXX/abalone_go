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
func (r *renderer) drawBoard(screen *ebiten.Image, g *board.Game, selPos int8) {
	screen.DrawImage(boardImg, nil)
	for pos := int8(0); pos < 61; pos++ {
		token := g.TokenAt(pos)
		if token == board.TokenEmpty || token == board.TokenVoid {
			continue
		}
		x := float64(cellCenters[pos][0] - 24)
		y := float64(cellCenters[pos][1] - 24)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		if token == board.PlayerA {
			screen.DrawImage(marbleAImg, op)
		} else {
			screen.DrawImage(marbleBImg, op)
		}
	}
	if selPos >= 0 {
		px := float64(cellCenters[selPos][0] - selHalfW)
		py := float64(cellCenters[selPos][1] - selHalfH)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(px, py)
		screen.DrawImage(selectedImg, op)
	}
	// 右侧被推出棋
	for p := 0; p < 2; p++ {
		for k := 0; k < outCounts[p]; k++ {
			ox := float64(outCoords[p][k][0] - 24)
			oy := float64(outCoords[p][k][1] - 24)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(ox, oy)
			if p == 0 {
				screen.DrawImage(marbleAImg, op)
			} else {
				screen.DrawImage(marbleBImg, op)
			}
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
