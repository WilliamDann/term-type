package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/rivo/tview"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: term-type [mode]

Modes:
  (none)       Open interactive menu
  time N       Timed mode (N = 15, 30, or 60 seconds)
  words N      Word count mode (N = 10, 25, or 50)
  history      Show history

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
		if err != nil || (n != 15 && n != 30 && n != 60) {
			fmt.Fprintf(os.Stderr, "Error: time must be 15, 30, or 60\n")
			os.Exit(1)
		}
		// Generate enough words for the time limit
		wc := map[int]int{15: 50, 30: 100, 60: 200}[n]
		return "test", true, n, wc
	case "words", "w":
		if len(args) < 2 {
			usage()
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || (n != 10 && n != 25 && n != 50) {
			fmt.Fprintf(os.Stderr, "Error: words must be 10, 25, or 50\n")
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

func main() {
	mode, argTimedMode, argTimeLimitSec, argWordCount := parseArgs()

	app := tview.NewApplication()
	pages := tview.NewPages()

	var (
		currentState *TestState
		ticker       *time.Ticker
		stopTimer    chan struct{}
	)

	// Forward declarations for mutual references
	var startTest func(timedMode bool, timeLimitSec int, wordCount int)
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
			startTest(currentState.TimedMode, currentState.TimeLimitSec, currentState.WordCount)
		}, showHistory)
		pages.AddAndSwitchToPage("results", resultsPage, true)
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
	case "history":
		showHistory()
	}

	app.SetRoot(pages, true)
	app.EnableMouse(false)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
