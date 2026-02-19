package main

import (
	"fmt"
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Braille dot positions: each character cell is 2 wide x 4 tall.
// Dot numbering:
//   0  3
//   1  4
//   2  5
//   6  7
var brailleDots = [4][2]rune{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

const brailleBase = '\u2800'

// WPMGraph is a tview primitive that renders a braille line graph of WPM over time.
type WPMGraph struct {
	*tview.Box
	snapshots []WPMSnapshot
}

func NewWPMGraph(snapshots []WPMSnapshot) *WPMGraph {
	return &WPMGraph{
		Box:       tview.NewBox(),
		snapshots: snapshots,
	}
}

func (g *WPMGraph) Draw(screen tcell.Screen) {
	g.Box.DrawForSubclass(screen, g)
	x, y, width, height := g.GetInnerRect()

	if len(g.snapshots) < 2 || width < 10 || height < 4 {
		return
	}

	// Reserve space for axis labels
	labelW := 5  // left Y-axis labels (e.g. " 120 ")
	bottomH := 1 // bottom X-axis labels

	graphX := x + labelW
	graphY := y
	graphW := width - labelW
	graphH := height - bottomH

	if graphW < 4 || graphH < 2 {
		return
	}

	// Compute data bounds for WPM
	minWPM, maxWPM := g.snapshots[0].WPM, g.snapshots[0].WPM
	maxTime := g.snapshots[len(g.snapshots)-1].Elapsed
	maxErrors := 0
	hasErrors := false
	for _, s := range g.snapshots {
		if s.WPM < minWPM {
			minWPM = s.WPM
		}
		if s.WPM > maxWPM {
			maxWPM = s.WPM
		}
		if s.Errors > maxErrors {
			maxErrors = s.Errors
		}
		if s.Errors > 0 {
			hasErrors = true
		}
	}

	// Add padding to WPM Y range
	wpmRange := maxWPM - minWPM
	if wpmRange < 10 {
		wpmRange = 10
		mid := (minWPM + maxWPM) / 2
		minWPM = mid - wpmRange/2
		maxWPM = mid + wpmRange/2
	}
	minWPM = math.Floor(minWPM/5) * 5
	maxWPM = math.Ceil(maxWPM/5) * 5
	if minWPM < 0 {
		minWPM = 0
	}
	wpmRange = maxWPM - minWPM

	// Error Y range (0 to maxErrors, minimum range of 1)
	errRange := float64(maxErrors)
	if errRange < 1 {
		errRange = 1
	}

	// Braille resolution: each cell = 2 cols x 4 rows of dots
	dotsW := graphW * 2
	dotsH := graphH * 4

	// Create braille grids (separate for WPM and errors so they get different colors)
	wpmGrid := make([][]rune, graphH)
	errGrid := make([][]rune, graphH)
	for i := range wpmGrid {
		wpmGrid[i] = make([]rune, graphW)
		errGrid[i] = make([]rune, graphW)
	}

	// Map WPM data points to dot positions
	type point struct{ dx, dy int }
	wpmPoints := make([]point, len(g.snapshots))
	errPoints := make([]point, len(g.snapshots))
	for i, s := range g.snapshots {
		fx := (s.Elapsed / maxTime) * float64(dotsW-1)
		// WPM: higher = top
		fy := (1.0 - (s.WPM-minWPM)/wpmRange) * float64(dotsH-1)
		wpmPoints[i] = point{clampInt(int(math.Round(fx)), 0, dotsW-1), clampInt(int(math.Round(fy)), 0, dotsH-1)}
		// Errors: higher = top (0 errors at bottom)
		ey := (1.0 - float64(s.Errors)/errRange) * float64(dotsH-1)
		errPoints[i] = point{clampInt(int(math.Round(fx)), 0, dotsW-1), clampInt(int(math.Round(ey)), 0, dotsH-1)}
	}

	// Draw WPM line
	for i := 0; i < len(wpmPoints)-1; i++ {
		plotBresenham(wpmGrid, wpmPoints[i].dx, wpmPoints[i].dy, wpmPoints[i+1].dx, wpmPoints[i+1].dy)
	}

	// Draw error line
	if hasErrors {
		for i := 0; i < len(errPoints)-1; i++ {
			plotBresenham(errGrid, errPoints[i].dx, errPoints[i].dy, errPoints[i+1].dx, errPoints[i+1].dy)
		}
	}

	// Render braille characters - errors first (underneath), then WPM on top
	errStyle := tcell.StyleDefault.Foreground(colorWrongFg).Background(colorBackground)
	lineStyle := tcell.StyleDefault.Foreground(colorAccent).Background(colorBackground)
	for row := 0; row < graphH; row++ {
		for col := 0; col < graphW; col++ {
			eCh := brailleBase + errGrid[row][col]
			wCh := brailleBase + wpmGrid[row][col]
			if wCh != brailleBase && eCh != brailleBase {
				// Both lines overlap in this cell - combine dots, WPM color wins
				combined := brailleBase + wpmGrid[row][col] | errGrid[row][col]
				screen.SetContent(graphX+col, graphY+row, combined, nil, lineStyle)
			} else if wCh != brailleBase {
				screen.SetContent(graphX+col, graphY+row, wCh, nil, lineStyle)
			} else if eCh != brailleBase {
				screen.SetContent(graphX+col, graphY+row, eCh, nil, errStyle)
			}
		}
	}

	// Draw Y-axis labels (WPM on left)
	axisStyle := tcell.StyleDefault.Foreground(colorSubtle).Background(colorBackground)
	topLabel := fmt.Sprintf("%3.0f ", maxWPM)
	midWPM := (minWPM + maxWPM) / 2
	midLabel := fmt.Sprintf("%3.0f ", midWPM)
	botLabel := fmt.Sprintf("%3.0f ", minWPM)

	drawString(screen, x, graphY, topLabel, axisStyle)
	if graphH > 2 {
		drawString(screen, x, graphY+graphH/2, midLabel, axisStyle)
	}
	drawString(screen, x, graphY+graphH-1, botLabel, axisStyle)

	// Draw X-axis labels
	startLabel := "0s"
	endLabel := fmt.Sprintf("%.0fs", maxTime)
	drawString(screen, graphX, graphY+graphH, startLabel, axisStyle)
	endX := graphX + graphW - len(endLabel)
	if endX > graphX+len(startLabel)+1 {
		drawString(screen, endX, graphY+graphH, endLabel, axisStyle)
	}

	// Draw legend on the bottom row, centered
	if hasErrors {
		legend := "── wpm  ── errors"
		legendX := graphX + (graphW-len(legend))/2
		if legendX < graphX {
			legendX = graphX
		}
		wpmLegendLine := "──"
		errLegendLine := "──"
		drawString(screen, legendX, graphY+graphH, wpmLegendLine, lineStyle)
		drawString(screen, legendX+len(wpmLegendLine), graphY+graphH, " wpm  ", axisStyle)
		drawString(screen, legendX+len(wpmLegendLine)+6, graphY+graphH, errLegendLine, errStyle)
		drawString(screen, legendX+len(wpmLegendLine)+6+len(errLegendLine), graphY+graphH, " errors", axisStyle)
	}
}

func drawString(screen tcell.Screen, x, y int, s string, style tcell.Style) {
	for i, ch := range s {
		screen.SetContent(x+i, y, ch, nil, style)
	}
}

func plotBresenham(grid [][]rune, x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	for {
		setBrailleDot(grid, x0, y0)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func setBrailleDot(grid [][]rune, dotX, dotY int) {
	cellCol := dotX / 2
	cellRow := dotY / 4
	if cellRow < 0 || cellRow >= len(grid) || cellCol < 0 || cellCol >= len(grid[0]) {
		return
	}
	subX := dotX % 2
	subY := dotY % 4
	grid[cellRow][cellCol] |= brailleDots[subY][subX]
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
