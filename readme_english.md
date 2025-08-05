**Abalone-Go**

A pure Go implementation of the Abalone board game engine with GUI.

---

## Resources

* **Game Rules & Reference**: [towzeur/gym-abalone](https://github.com/towzeur/gym-abalone)
* **Search Algorithm Papers**:

  * *A Critical Review – Exploring Optimization Strategies in Board Game Abalone for Alpha-Beta Search* (Radboud, 2021)
  * *Abalone AI Final Report* (Cornell CS, 2020)

---

## Directory Structure

```
abalone_go/
├─ cmd/abalone/        # Main executable
├─ internal/
│   ├─ board/          # Game rules, Zobrist hashing
│   ├─ search/         # PVS, NullMove, LMR, Transposition Table, static move ordering
│   ├─ eval/           # Evaluation function
│   ├─ ui/             # Ebiten rendering & input handling
│   └─ ...
└─ README.md
```

---

## Installation & Running

```bash
# Requires Go 1.22+
git clone https://github.com/H1W0XXX/abalone-go
cd abalone-go

go vet ./...
go build -o abalone ./cmd/abalone

# Play against AI (depth 4 recommended; avoid depth >5)
./abalone -mode=pve -depth=4

# Two-player mode (same screen)
./abalone -mode=pvp

# Randomize first player
./abalone -mode=pve -depth=5 -random
```

**Controls**

* Press `Esc` to quit
* Click your marble, then click the target cell to move

---

## Engine Features

| Feature           | Description                                                                                         |
| ----------------- | --------------------------------------------------------------------------------------------------- |
| **Search**        | PVS, Null-Move (R=2), LMR, static move ordering                                                     |
| **Transposition** | 64-bit Zobrist hashing + transposition table                                                        |
| **Evaluation**    | Center distance (h₁) + connectivity (h₂) + marble count (h₃) + edge penalty + push potential reward |
| **Concurrency**   | Root-node parallelism using N-1 goroutines                                                          |
| **GUI**           | Ebiten at 60 FPS with embedded static assets                                                        |

*On an Intel Core i9-14900K, the engine searches 4 plies in \~2s, matching the strongest published results.*

---

## CLI Options

| Option    | Default | Description                               |
| --------- | ------- | ----------------------------------------- |
| `-mode`   | `pve`   | `pve` (AI opponent) or `pvp` (two-player) |
| `-depth`  | `4`     | Fixed search depth                        |
| `-random` | `false` | Randomize who moves first                 |
