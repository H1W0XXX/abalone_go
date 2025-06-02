package ui

import (
	_ "embed"
)

// 这里我们已经在 renderer.go 里加载了 marbleAImg, marbleBImg, arrowAImg, arrowBImg, selectedImg
// marbleUI 只负责当前棋子的“sprite状态”（position, direction, 是否选中）
// 其实不用额外存一份 sprite，在 renderer.Draw 时只需根据逻辑层的内容重绘即可。
// 所以这个文件可以略简化，直接让 renderer 取 token、selPos、arrowVisible 去 draw 即可。
// 这里只说明思路，不给完整实现。

type marbleUI struct {
	pos       int8 // 0..60
	player    int8 // 0=A, 1=B
	direction int8 // 0..5，如果 <0 表示箭头不显示
	selected  bool
	outIndex  int // 如果被 Eject，要移动到屏幕外
}

// 直接写一个 draw()：
// func (m *marbleUI) draw(screen *ebiten.Image) {
//    if m.outIndex >= 0 {
//       x,y := outCoordinates[m.player][m.outIndex]
//       draw marbleImg at (x,y)
//       return
//    }
//    centerX := cellCenters[m.pos][0]
//    centerY := cellCenters[m.pos][1]
//    // draw marble 本体
//    img := marbleAImg if m.player==0 else marbleBImg
//    op := &ebiten.DrawImageOptions{}
//    op.GeoM.Translate(float64(centerX-24), float64(centerY-24))
//    screen.DrawImage(img, op)
//    // draw arrow
//    if m.direction >= 0 {
//       arrowImg := arrowAImg if m.player==0 else arrowBImg
//       op2 := &ebiten.DrawImageOptions{}
//       op2.GeoM.Translate(float64(centerX-24), float64(centerY-24))
//       op2.GeoM.Rotate(float64(m.direction) * (2*math.Pi/6)) // 60度一次
//       screen.DrawImage(arrowImg, op2)
//    }
//    // draw selected
//    if m.selected {
//       op3 := &ebiten.DrawImageOptions{}
//       op3.GeoM.Translate(float64(centerX-24), float64(centerY-24))
//       screen.DrawImage(selectedImg, op3)
//    }
// }
