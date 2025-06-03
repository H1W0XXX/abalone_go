package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"abalone_go/internal/board"
	"abalone_go/internal/ui"
)

func main() {
	// ──────── 命令行参数 ────────
	var (
		randomStart = flag.Bool("random", false, "randomize starting player")
		maxDepth    = flag.Int("depth", 4, "search depth for AI")
		mode        = flag.String("mode", "pve", "game mode: pve or pvp")
	)
	flag.Parse()

	// ──────── 初始化棋局 ────────
	startPlayer := board.PlayerA
	if *randomStart {
		rand.Seed(time.Now().UnixNano())
		startPlayer = board.PlayerA + int8(rand.Intn(2))
	}
	g := board.NewGame(startPlayer)

	fmt.Printf("Abalone started: first move -> Player %d  |  mode=%s  |  depth=%d\n",
		startPlayer, *mode, *maxDepth)

	// ──────── 启动 UI 主循环 ────────
	pve := (*mode == "pve") // true = 双人
	gameLoop := ui.NewGameLoop(g, pve, int8(*maxDepth))
	ui.Run(gameLoop)
}

// go build -ldflags="-s -w" -gcflags="all=-trimpath=${PWD}" -asmflags="all=-trimpath=${PWD}" -o abalone.exe .\cmd\abalone\main.go
