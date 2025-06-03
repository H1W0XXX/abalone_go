# Abalone-Go

**纯 Go 实现的 Abalone（六珠棋）引擎 + GUI**

* 游戏规则与资源：https://github.com/towzeur/gym-abalone
* 搜索算法：基于两篇论文

  * 《A Critical Review – Exploring Optimization Strategies in Board Game Abalone for Alpha-Beta Search》（Radboud，2021）
  * 《Abalone AI Final Report》（Cornell CS，2020）

## 目录结构

```
abalone_go/
├─ cmd/abalone/        主程序
├─ internal/
│   ├─ board/          规则、Zobrist
│   ├─ search/         PVS + NullMove + LMR + TT + 静态排序
│   ├─ eval/           评估函数
│   ├─ ui/             Ebiten 渲染与输入
│   └─ ...
└─ README.md
```

## 编译运行

```bash
# 依赖 Go 1.22+
git clone https://github.com/yourname/abalone-go
cd abalone-go
go vet ./...
go build -o abalone ./cmd/abalone

./abalone -mode=pve -depth=4     # 人机对战，不建议超过5
./abalone -mode=pvp              # 双人同屏
./abalone -mode=pve -depth=5 -random   # 随机先手
```

* `Esc` 退出
* 点击己子再点击目标完成落子

## 引擎特性

| 项   | 说明                                       |
| --- | ---------------------------------------- |
| 搜索  | PVS、Null-Move (R=2)、LMR、静态排序             |
| 局面库 | 64 位 Zobrist + 置换表                       |
| 评估  | 中心距离 h₁ + 连通块 h₂ + 子数 h₃ + 边缘惩罚 + 潜在推子奖励 |
| 多核  | 根节点 N-1 goroutine 并行                     |
| GUI | Ebiten 60 FPS，静态资源内嵌                     |

在 14900k 上 2 s 可搜索 4 ply，棋力与论文最佳结果相当。

## CLI 参数

| 参数        | 默认      | 说明          |
| --------- | ------- | ----------- |
| `-mode`   | `pve`   | `pve`/`pvp` |
| `-depth`  | `4`     | 固定搜索深度      |
| `-random` | `false` | 随机先手        |
