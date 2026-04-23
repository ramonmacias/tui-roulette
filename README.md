# TUI Roulette

```text
██████╗  ██████╗ ██╗   ██╗██╗     ███████╗████████╗████████╗███████╗
██╔══██╗██╔═══██╗██║   ██║██║     ██╔════╝╚══██╔══╝╚══██╔══╝██╔════╝
██████╔╝██║   ██║██║   ██║██║     █████╗     ██║      ██║   █████╗  
██╔══██╗██║   ██║██║   ██║██║     ██╔══╝     ██║      ██║   ██╔══╝  
██║  ██║╚██████╔╝╚██████╔╝███████╗███████╗   ██║      ██║   ███████╗
╚═╝  ╚═╝ ╚═════╝  ╚═════╝ ╚══════╝╚══════╝   ╚═╝      ╚═╝   ╚══════╝
```

A terminal-first roulette manager built with Go + Bubble Tea.

It supports multiple spin modes, animated spins, participant enable/disable, and local persistence.

---

## Features

- Full-screen interactive TUI
- Animated roulette spin popup
- Multiple modes:
  - Repeatable winners
  - No-repeat winners
  - Elimination
  - Multi-winner
- Participant management:
  - Add/remove participants
  - Enable/disable participants (affects candidate pool)
- Persistent local storage (JSON file)
- Keyboard-driven workflow

---

## Requirements

- Go 1.24+ (project currently uses Go 1.24.4)
- A terminal with ANSI color support

---

## Installation

### 1) Clone the repository

```bash
git clone <your-repo-url>
cd tui-roulette
```

### 2) Download dependencies

```bash
go mod download
```

(If you vendor dependencies, this may not be necessary.)

### 3) Build

```bash
go build ./...
```

---

## Run

### Run directly

```bash
go run .
```

### Or run built binary

```bash
./tui-roulette
```

(If your binary name differs, run the generated executable accordingly.)

---

## How it works

The app opens in an alternate full-screen TUI.

### Core concepts

- **Participants**: all people added to a roulette
- **Candidates**: participants currently eligible to be selected on spin
  - Candidates exclude disabled participants
  - Depending on mode, winners/eliminated participants may also be excluded

### Roulette modes

1. **Repeatable winners**
   - Any candidate can win repeatedly.

2. **No-repeat winners**
   - Winners are excluded from future candidate picks.

3. **Elimination**
   - Spins eliminate participants one by one until one winner remains.

4. **Multi-winner**
   - A spin selects multiple winners in one run (based on configured winner count).

---

## Keyboard shortcuts

### Browse mode

- `n` → New roulette
- `a` → Add participant
- `d` / `x` / `backspace` → Remove selected participant
- `space` → Enable/disable selected participant
- `s` → Spin roulette (opens spin roulette popup)
- `r` → Reset current roulette winners/eliminated state
- `tab` / `←` / `→` → Switch focus (roulettes/participants)
- `↑` / `↓` (or `k` / `j`) → Move selection
- `q` → Quit

### Spin roulette popup

- `s` → Spin again
- `esc` → Close popup

### Global

- `ctrl+c` → Quit

---

## Persistence

Data is stored in:

- `~/.config/tui-roulette/roulettes.json`

Saved data includes roulette definitions, participants, winners, eliminated participants, disabled participants, mode, and multi-winner settings.

---

## Run tests

Run all tests:

```bash
go test ./...
```

Run verbose tests:

```bash
go test -v ./...
```

Run tests for internal package only:

```bash
go test -v ./internal
```

---

## Contributing

Contributions are welcome.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run formatting and tests
5. Open a pull request

### Suggested local checks before PR

```bash
go fmt ./...
go test ./...
go build ./...
```

Please keep changes focused, maintain current behavior unless explicitly changing it, and include/update tests for new functionality.

---

## License

See [LICENSE](LICENSE).
