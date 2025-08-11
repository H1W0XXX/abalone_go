package ui

import "github.com/hajimehoshi/ebiten/v2"

var (
	perfOn   bool // 是否处于高性能模式
	booted   bool // 首帧已启动（避免在首帧前切模式）
	animBusy int  // 正在播放的动画计数（用作引用计数）
)

func enterPerf() {
	if !booted || perfOn {
		return
	}
	ebiten.SetMaxTPS(60)                     // 动画更顺
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOn) // 或 VsyncOffMaximum
	perfOn = true
}
func leavePerf() {
	if !booted || !perfOn {
		return
	}
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMinimum) // 省电
	ebiten.SetMaxTPS(10)
	perfOn = false
}
