package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: term-type [mode]

Modes:
  (none)       Open interactive menu
  time N       Timed mode (N seconds)
  words N      Word count mode (N words)
  history      Show history

Piped input:
  echo "custom text" | term-type
  cat quote.txt | term-type

Examples:
  term-type
  term-type time 30
  term-type words 25
  term-type history
`)
	os.Exit(1)
}

func parseArgs() (mode string, timedMode bool, timeLimitSec int, wordCount int) {
	args := os.Args[1:]
	if len(args) == 0 {
		return "menu", false, 0, 0
	}

	switch args[0] {
	case "time", "t":
		if len(args) < 2 {
			usage()
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "Error: time must be a positive number of seconds\n")
			os.Exit(1)
		}
		// Generate roughly 3-4 words per second of typing
		wc := n * 4
		return "test", true, n, wc
	case "words", "w":
		if len(args) < 2 {
			usage()
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "Error: words must be a positive number\n")
			os.Exit(1)
		}
		return "test", false, 0, n
	case "history", "h":
		return "history", false, 0, 0
	case "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", args[0])
		usage()
	}
	return "menu", false, 0, 0
}

// readPipedInput reads from stdin if it's a pipe, normalizes whitespace,
// then reopens /dev/tty so tcell can read keyboard input.
func readPipedInput() (string, bool) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", false
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		// stdin is a terminal, not a pipe
		return "", false
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(data) == 0 {
		return "", false
	}

	// Normalize: collapse all whitespace into single spaces, trim
	text := strings.TrimSpace(string(data))
	var b strings.Builder
	inSpace := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteRune(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	text = b.String()
	if text == "" {
		return "", false
	}

	// Reopen /dev/tty as stdin so tcell gets keyboard input
	tty, err := os.Open("/dev/tty")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot open /dev/tty for keyboard input: %v\n", err)
		os.Exit(1)
	}
	os.Stdin = tty

	return text, true
}

func main() {
	pipedText, hasPiped := readPipedInput()
	mode, argTimedMode, argTimeLimitSec, argWordCount := parseArgs()

	// Piped input overrides mode
	if hasPiped {
		mode = "pipe"
	}

	app := tview.NewApplication()

	// When stdin was a pipe, tell tcell to use /dev/tty directly
	if hasPiped {
		screen, err := tcell.NewScreen()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating screen: %v\n", err)
			os.Exit(1)
		}
		app.SetScreen(screen)
	}

	pages := tview.NewPages()

	var (
		currentState *TestState
		ticker       *time.Ticker
		stopTimer    chan struct{}
	)

	// Forward declarations for mutual references
	var startTest func(timedMode bool, timeLimitSec int, wordCount int)
	var startTestWithText func(text string)
	var showResults func()
	var showHistory func()

	showHistory = func() {
		histPage := buildHistory(app, pages)
		pages.AddAndSwitchToPage("history", histPage, true)
	}

	showResults = func() {
		if currentState == nil {
			return
		}
		currentState.Finish()

		// Save result
		_ = saveResult(Result{
			Date:     time.Now(),
			Mode:     currentState.ModeString(),
			WPM:      math.Round(currentState.WPM()),
			Accuracy: currentState.Accuracy(),
			Correct:  currentState.CorrectChars(),
			Wrong:    currentState.WrongChars(),
		})

		resultsPage := buildResults(app, pages, currentState, func() {
			// Retry with same settings
			if currentState.PipedText != "" {
				startTestWithText(currentState.PipedText)
			} else {
				startTest(currentState.TimedMode, currentState.TimeLimitSec, currentState.WordCount)
			}
		}, showHistory)
		pages.AddAndSwitchToPage("results", resultsPage, true)
	}

	// startTestWithText starts a typing test using provided text (for piped input)
	startTestWithText = func(text string) {
		wordCount := len(strings.Fields(text))
		currentState = NewTestState(text, false, 0, wordCount)
		currentState.PipedText = text

		onFinish := func() {
			showResults()
		}
		onEscape := func() {
			pages.SwitchToPage("menu")
		}

		typingBox := NewTypingBox(currentState, onFinish, onEscape)
		pages.AddAndSwitchToPage("typing", typingBox, true)
	}

	startTest = func(timedMode bool, timeLimitSec int, wordCount int) {
		// Stop any existing timer
		if stopTimer != nil {
			close(stopTimer)
			stopTimer = nil
		}
		if ticker != nil {
			ticker.Stop()
			ticker = nil
		}

		target := pickWords(wordCount)
		currentState = NewTestState(target, timedMode, timeLimitSec, wordCount)

		onFinish := func() {
			if stopTimer != nil {
				close(stopTimer)
				stopTimer = nil
			}
			if ticker != nil {
				ticker.Stop()
				ticker = nil
			}
			showResults()
		}

		onEscape := func() {
			if stopTimer != nil {
				close(stopTimer)
				stopTimer = nil
			}
			if ticker != nil {
				ticker.Stop()
				ticker = nil
			}
			pages.SwitchToPage("menu")
		}

		typingBox := NewTypingBox(currentState, onFinish, onEscape)

		pages.AddAndSwitchToPage("typing", typingBox, true)

		if timedMode {
			// Start a goroutine that watches for the test to start, then counts down
			stopTimer = make(chan struct{})
			go func(state *TestState, stop chan struct{}) {
				// Wait for test to start
				for !state.Started {
					select {
					case <-stop:
						return
					case <-time.After(50 * time.Millisecond):
					}
				}

				// Start the countdown ticker
				t := time.NewTicker(100 * time.Millisecond)
				defer t.Stop()

				for {
					select {
					case <-stop:
						return
					case <-t.C:
						if state.Finished {
							return
						}
						if state.TimeRemaining() <= 0 {
							app.QueueUpdateDraw(func() {
								if !state.Finished {
									onFinish()
								}
							})
							return
						}
						app.QueueUpdateDraw(func() {})
					}
				}
			}(currentState, stopTimer)
		}
	}

	menu := buildMenu(app, pages, startTest, showHistory)
	pages.AddPage("menu", menu, true, true)

	switch mode {
	case "test":
		startTest(argTimedMode, argTimeLimitSec, argWordCount)
	case "pipe":
		startTestWithText(pipedText)
	case "history":
		showHistory()
	}

	app.SetRoot(pages, true)
	app.EnableMouse(false)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
