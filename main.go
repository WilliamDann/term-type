package main

import (
	"math"
	"time"

	"github.com/rivo/tview"
)

func main() {
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

	app.SetRoot(pages, true)
	app.EnableMouse(false)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
