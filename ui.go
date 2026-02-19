package main

import (
	"fmt"
	"math"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func buildMenu(app *tview.Application, pages *tview.Pages, startTest func(timedMode bool, timeLimitSec int, wordCount int), showHistory func(), showThemes func()) *tview.Flex {
	list := tview.NewList().
		AddItem("Time 15s", "Timed mode - 15 seconds", '1', func() {
			startTest(true, 15, 50)
		}).
		AddItem("Time 30s", "Timed mode - 30 seconds", '2', func() {
			startTest(true, 30, 100)
		}).
		AddItem("Time 60s", "Timed mode - 60 seconds", '3', func() {
			startTest(true, 60, 200)
		}).
		AddItem("Words 10", "Type 10 words", '4', func() {
			startTest(false, 0, 10)
		}).
		AddItem("Words 25", "Type 25 words", '5', func() {
			startTest(false, 0, 25)
		}).
		AddItem("Words 50", "Type 50 words", '6', func() {
			startTest(false, 0, 50)
		}).
		AddItem("History", "View past results", 'h', func() {
			showHistory()
		}).
		AddItem("Theme", "Change color theme", 't', func() {
			showThemes()
		}).
		AddItem("Quit", "Exit the application", 'q', func() {
			app.Stop()
		})

	list.SetBackgroundColor(colorBackground)
	list.SetMainTextColor(colorCorrect)
	list.SetSecondaryTextColor(colorSubtle)
	list.SetSelectedTextColor(colorBackground)
	list.SetSelectedBackgroundColor(colorAccent)
	list.SetShortcutColor(colorAccent)

	title := tview.NewTextView().
		SetText("term-type").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorAccent)
	title.SetBackgroundColor(colorBackground)

	subtitle := tview.NewTextView().
		SetText("A typing speed test for your terminal").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorSubtle)
	subtitle.SetBackgroundColor(colorBackground)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(title, 1, 0, false).
		AddItem(subtitle, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(list, 40, 0, true).
			AddItem(nil, 0, 1, false),
			0, 1, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(colorBackground)

	return flex
}

func buildResults(app *tview.Application, pages *tview.Pages, state *TestState, onRetry func(), onHistory func()) *tview.Flex {
	wpm := math.Round(state.WPM())
	acc := state.Accuracy()
	correct := state.CorrectChars()
	wrong := state.WrongChars()

	wpmView := tview.NewTextView().
		SetText(fmt.Sprintf("%.0f", wpm)).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorAccent)
	wpmView.SetBackgroundColor(colorBackground)

	wpmLabel := tview.NewTextView().
		SetText("wpm").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorSubtle)
	wpmLabel.SetBackgroundColor(colorBackground)

	accView := tview.NewTextView().
		SetText(fmt.Sprintf("%.1f%%", acc)).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorAccent)
	accView.SetBackgroundColor(colorBackground)

	accLabel := tview.NewTextView().
		SetText("accuracy").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorSubtle)
	accLabel.SetBackgroundColor(colorBackground)

	statsView := tview.NewTextView().
		SetText(fmt.Sprintf("%d correct  /  %d wrong  /  %s", correct, wrong, state.ModeString())).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorSubtle)
	statsView.SetBackgroundColor(colorBackground)

	helpView := tview.NewTextView().
		SetText("[enter] retry  [tab] menu  [h] history  [q] quit").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorSubtle)
	helpView.SetBackgroundColor(colorBackground)

	// WPM graph
	graph := NewWPMGraph(state.WPMSnapshots)
	graph.SetBackgroundColor(colorBackground)

	graphWrapper := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(graph, 60, 0, false).
		AddItem(nil, 0, 1, false)
	graphWrapper.SetBackgroundColor(colorBackground)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(wpmView, 2, 0, false).
		AddItem(wpmLabel, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(accView, 2, 0, false).
		AddItem(accLabel, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(statsView, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(graphWrapper, 0, 2, false).
		AddItem(nil, 1, 0, false).
		AddItem(helpView, 1, 0, true).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(colorBackground)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			onRetry()
			return nil
		case tcell.KeyTab:
			pages.SwitchToPage("menu")
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'h':
				onHistory()
				return nil
			case 'q':
				app.Stop()
				return nil
			}
		}
		return event
	})

	return flex
}

func buildHistory(app *tview.Application, pages *tview.Pages, onClear ...func()) *tview.Flex {
	table := tview.NewTable().
		SetFixed(1, 0).
		SetSelectable(true, false)
	table.SetBackgroundColor(colorBackground)
	table.SetSelectedStyle(tcell.StyleDefault.Background(colorAccent).Foreground(colorBackground))

	// Headers
	headers := []string{"Date", "Mode", "WPM", "Accuracy"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(colorAccent).
			SetSelectable(false).
			SetExpansion(1).
			SetAlign(tview.AlignCenter)
		table.SetCell(0, i, cell)
	}

	results, _ := loadHistory()
	// Show newest first
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	for i, r := range results {
		row := i + 1
		table.SetCell(row, 0, tview.NewTableCell(r.Date.Format(time.DateTime)).
			SetTextColor(colorCorrect).SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(row, 1, tview.NewTableCell(r.Mode).
			SetTextColor(colorCorrect).SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.0f", r.WPM)).
			SetTextColor(colorCorrect).SetAlign(tview.AlignCenter).SetExpansion(1))
		table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%.1f%%", r.Accuracy)).
			SetTextColor(colorCorrect).SetAlign(tview.AlignCenter).SetExpansion(1))
	}

	if len(results) == 0 {
		table.SetCell(1, 0, tview.NewTableCell("No results yet").
			SetTextColor(colorSubtle).SetAlign(tview.AlignCenter).SetExpansion(4))
	}

	title := tview.NewTextView().
		SetText("History").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorAccent)
	title.SetBackgroundColor(colorBackground)

	helpText := "[esc] back to menu"
	if len(results) > 0 && len(onClear) > 0 {
		helpText = "[esc] back to menu  [c] clear history"
	}
	helpView := tview.NewTextView().
		SetText(helpText).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorSubtle)
	helpView.SetBackgroundColor(colorBackground)

	lpad := tview.NewBox().SetBackgroundColor(colorBackground)
	rpad := tview.NewBox().SetBackgroundColor(colorBackground)
	innerFlex := tview.NewFlex().
		AddItem(lpad, 2, 0, false).
		AddItem(table, 0, 1, true).
		AddItem(rpad, 2, 0, false)
	innerFlex.SetBackgroundColor(colorBackground)

	topPad := tview.NewBox().SetBackgroundColor(colorBackground)
	midPad := tview.NewBox().SetBackgroundColor(colorBackground)
	botPad := tview.NewBox().SetBackgroundColor(colorBackground)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topPad, 1, 0, false).
		AddItem(title, 1, 0, false).
		AddItem(midPad, 1, 0, false).
		AddItem(innerFlex, 0, 1, true).
		AddItem(helpView, 1, 0, false).
		AddItem(botPad, 1, 0, false)
	flex.SetBackgroundColor(colorBackground)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("menu")
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'c' && len(onClear) > 0 {
			clearHistory()
			onClear[0]()
			return nil
		}
		return event
	})

	return flex
}

func buildThemePicker(app *tview.Application, pages *tview.Pages, onSelect func(string)) *tview.Flex {
	list := tview.NewList()

	for i, name := range themeOrder {
		shortcut := rune('a' + i)
		n := name // capture for closure
		list.AddItem(name, "", shortcut, func() {
			onSelect(n)
		})
	}

	list.SetBackgroundColor(colorBackground)
	list.SetMainTextColor(colorCorrect)
	list.SetSecondaryTextColor(colorSubtle)
	list.SetSelectedTextColor(colorBackground)
	list.SetSelectedBackgroundColor(colorAccent)
	list.SetShortcutColor(colorAccent)

	title := tview.NewTextView().
		SetText("Theme").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorAccent)
	title.SetBackgroundColor(colorBackground)

	helpView := tview.NewTextView().
		SetText("[esc] back to menu").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(colorSubtle)
	helpView.SetBackgroundColor(colorBackground)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(title, 1, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(list, 40, 0, true).
			AddItem(nil, 0, 1, false),
			0, 1, true).
		AddItem(helpView, 1, 0, false).
		AddItem(nil, 0, 1, false)
	flex.SetBackgroundColor(colorBackground)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("menu")
			return nil
		}
		return event
	})

	return flex
}
