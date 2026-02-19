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
	fmt.Fprintf(os.Stderr, `Usage: term-type [--theme NAME] [mode]

Modes:
  (none)       Open interactive menu
  time N       Timed mode (N seconds)
  words N      Word count mode (N words)
  history      Show history
  clear history  Clear history
  clear theme    Reset theme to default
  themes       List available themes

Options:
  --theme NAME   Set color theme (auto-detects Omarchy theme by default)

Piped input:
  echo "custom text" | term-type
  cat quote.txt | term-type

Examples:
  term-type
  term-type --theme catppuccin time 30
  term-type words 25
  term-type themes
`)
	os.Exit(1)
}

func parseArgs() (mode string, timedMode bool, timeLimitSec int, wordCount int, themeName string) {
	args := os.Args[1:]

	// Extract --theme flag
	var filtered []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--theme" {
			if i+1 < len(args) {
				themeName = args[i+1]
				i++ // skip value
			} else {
				fmt.Fprintf(os.Stderr, "Error: --theme requires a theme name\n")
				os.Exit(1)
			}
		} else {
			filtered = append(filtered, args[i])
		}
	}
	args = filtered

	if len(args) == 0 {
		return "menu", false, 0, 0, themeName
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
		return "test", true, n, wc, themeName
	case "words", "w":
		if len(args) < 2 {
			usage()
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "Error: words must be a positive number\n")
			os.Exit(1)
		}
		return "test", false, 0, n, themeName
	case "history", "h":
		return "history", false, 0, 0, themeName
	case "themes":
		fmt.Println("Available themes:")
		for _, name := range themeOrder {
			fmt.Printf("  %s\n", name)
		}
		detected := detectOmarchyTheme()
		if detected != "" {
			fmt.Printf("\nCurrent Omarchy theme: %s\n", detected)
		}
		os.Exit(0)
	case "clear":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: term-type clear <history|theme>\n")
			os.Exit(1)
		}
		switch args[1] {
		case "history":
			if err := clearHistory(); err != nil {
				fmt.Fprintf(os.Stderr, "Error clearing history: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("History cleared.")
		case "theme":
			if err := clearThemePreference(); err != nil {
				fmt.Fprintf(os.Stderr, "Error clearing theme: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Theme reset to default.")
		default:
			fmt.Fprintf(os.Stderr, "Unknown clear target: %s\n", args[1])
			fmt.Fprintf(os.Stderr, "Usage: term-type clear <history|theme>\n")
			os.Exit(1)
		}
		os.Exit(0)
	case "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", args[0])
		usage()
	}
	return "menu", false, 0, 0, themeName
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
	mode, argTimedMode, argTimeLimitSec, argWordCount, themeName := parseArgs()

	initTheme(themeName)

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
	var showThemes func()
	var rebuildMenu func()

	rebuildMenu = func() {
		menu := buildMenu(app, pages, startTest, showHistory, showThemes)
		pages.AddAndSwitchToPage("menu", menu, true)
	}

	showThemes = func() {
		picker := buildThemePicker(app, pages, func(name string) {
			initTheme(name)
			saveThemePreference(name)
			rebuildMenu()
		})
		pages.AddAndSwitchToPage("themes", picker, true)
	}

	showHistory = func() {
		histPage := buildHistory(app, pages, func() {
			showHistory()
		})
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

	menu := buildMenu(app, pages, startTest, showHistory, showThemes)
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
