# term-type

A monkeytype-inspired typing speed test for your terminal, built with Go and [tview](https://github.com/rivo/tview).

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)

## Features

- **6 modes** — Timed (15s, 30s, 60s) and word count (10, 25, 50)
- **Live WPM** — Real-time words-per-minute display while typing
- **Per-character feedback** — Correct, wrong, and pending characters colored distinctly
- **History** — Results saved to `~/.local/share/term-type/history.json`
- **Monkeytype palette** — Serika Dark color scheme

## Install

```bash
go install github.com/you/term-type@latest
```

Or build from source:

```bash
git clone https://github.com/you/term-type.git
cd term-type
go install .
```

## Usage

```
term-type
```

### Controls

| Key | Action |
|---|---|
| `1`-`6` | Select mode from menu |
| Any key | Type (timer starts on first keypress) |
| `Backspace` | Delete last character |
| `Ctrl+W` | Delete last word |
| `Escape` | Return to menu |
| `Enter` | Retry (on results screen) |
| `Tab` | Back to menu (on results screen) |
| `h` | View history |
| `q` | Quit |

## How WPM is calculated

- **WPM**: `(correct characters / 5) / elapsed minutes`
- **Accuracy**: `correct characters / total typed * 100`

This matches the standard formula used by monkeytype and other typing tests.
