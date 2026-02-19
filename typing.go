package main

import (
	"fmt"
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Serika Dark color palette
var (
	colorPending    = tcell.NewRGBColor(0x64, 0x66, 0x69)
	colorCorrect    = tcell.NewRGBColor(0xD1, 0xD0, 0xC5)
	colorWrongFg    = tcell.NewRGBColor(0xCA, 0x47, 0x54)
	colorWrongBg    = tcell.NewRGBColor(0x3A, 0x1C, 0x20)
	colorCursor     = tcell.NewRGBColor(0xE2, 0xB7, 0x14)
	colorAccent     = tcell.NewRGBColor(0xE2, 0xB7, 0x14)
	colorBackground = tcell.NewRGBColor(0x32, 0x36, 0x37)
	colorSubtle     = tcell.NewRGBColor(0x64, 0x66, 0x69)
)

type TypingBox struct {
	*tview.Box
	state    *TestState
	onFinish func()
	onEscape func()
}

func NewTypingBox(state *TestState, onFinish func(), onEscape func()) *TypingBox {
	tb := &TypingBox{
		Box:      tview.NewBox(),
		state:    state,
		onFinish: onFinish,
		onEscape: onEscape,
	}
	tb.SetBackgroundColor(colorBackground)
	return tb
}

func (t *TypingBox) Draw(screen tcell.Screen) {
	t.Box.DrawForSubclass(screen, t)
	x, y, width, height := t.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}

	target := []rune(t.state.Target)
	input := t.state.Input
	cursorPos := len(input)

	// Word-wrap: break target text into lines that fit within width
	type lineInfo struct {
		start int // index into target runes
		end   int // exclusive
	}
	var lines []lineInfo

	i := 0
	for i < len(target) {
		lineStart := i
		lastSpace := -1
		col := 0
		for i < len(target) && col < width {
			if target[i] == ' ' {
				lastSpace = i
			}
			col++
			i++
		}
		if i < len(target) && lastSpace > lineStart {
			// Wrap at last space
			lines = append(lines, lineInfo{lineStart, lastSpace + 1})
			i = lastSpace + 1
		} else {
			lines = append(lines, lineInfo{lineStart, i})
		}
	}

	if len(lines) == 0 {
		return
	}

	// Find which line the cursor is on to determine scroll offset
	cursorLine := 0
	for li, ln := range lines {
		if cursorPos >= ln.start && cursorPos < ln.end {
			cursorLine = li
			break
		}
		if cursorPos >= ln.end {
			cursorLine = li + 1
		}
	}

	// Show a few lines of context, centered around cursor line
	// Reserve top line for timer/info
	infoY := y
	textStartY := y + 2
	maxTextLines := height - 3

	if maxTextLines < 1 {
		maxTextLines = 1
	}

	// Scroll so cursor line is visible
	scrollOffset := 0
	if cursorLine >= maxTextLines {
		scrollOffset = cursorLine - maxTextLines/2
		if scrollOffset < 0 {
			scrollOffset = 0
		}
	}

	// Draw timer/info line
	var info string
	if t.state.TimedMode {
		remaining := t.state.TimeRemaining()
		info = fmt.Sprintf("%.1f", remaining)
	} else {
		// Show word progress
		wordsTyped := 0
		for _, ch := range input {
			if ch == ' ' {
				wordsTyped++
			}
		}
		if t.state.Finished {
			wordsTyped = t.state.WordCount
		}
		info = fmt.Sprintf("%d/%d", wordsTyped, t.state.WordCount)
	}
	infoStyle := tcell.StyleDefault.Background(colorBackground).Foreground(colorAccent).Bold(true)
	infoX := x + (width-len(info))/2
	for ci, ch := range info {
		screen.SetContent(infoX+ci, infoY, ch, nil, infoStyle)
	}

	// Draw each visible line
	for li := scrollOffset; li < len(lines) && li-scrollOffset < maxTextLines; li++ {
		ln := lines[li]
		lineY := textStartY + (li - scrollOffset)

		// Center the line
		lineLen := ln.end - ln.start
		lineX := x + (width-lineLen)/2

		for ci := ln.start; ci < ln.end; ci++ {
			ch := target[ci]
			style := tcell.StyleDefault.Background(colorBackground)

			if ci < cursorPos {
				// Already typed
				if ci < len(input) && input[ci] == target[ci] {
					style = style.Foreground(colorCorrect)
				} else {
					style = style.Foreground(colorWrongFg).Background(colorWrongBg)
					// Show what user typed if it's a printable char, otherwise show target
					if ci < len(input) && input[ci] != target[ci] {
						ch = target[ci] // Keep target char visible but colored wrong
					}
				}
			} else if ci == cursorPos {
				// Cursor position
				style = style.Foreground(colorCursor).Underline(true)
			} else {
				// Pending
				style = style.Foreground(colorPending)
			}

			screen.SetContent(lineX+(ci-ln.start), lineY, ch, nil, style)
		}
	}

	// Draw WPM at bottom if test is in progress and user has started typing
	if t.state.Started && !t.state.Finished {
		wpm := t.state.WPM()
		wpmStr := fmt.Sprintf("%.0f wpm", math.Round(wpm))
		wpmStyle := tcell.StyleDefault.Background(colorBackground).Foreground(colorSubtle)
		wpmX := x + (width-len(wpmStr))/2
		wpmY := y + height - 1
		for ci, ch := range wpmStr {
			screen.SetContent(wpmX+ci, wpmY, ch, nil, wpmStyle)
		}
	}
}

func (t *TypingBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return t.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if t.state.Finished {
			return
		}

		switch event.Key() {
		case tcell.KeyEscape:
			t.onEscape()
			return
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			t.state.HandleBackspace()
			return
		case tcell.KeyCtrlW:
			t.state.HandleDeleteWord()
			return
		case tcell.KeyRune:
			t.state.HandleChar(event.Rune())
			if t.state.Finished {
				t.onFinish()
			}
			return
		}
	})
}
