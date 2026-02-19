package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type themeColors struct {
	background string
	foreground string
	accent     string
	cursor     string
	errColor   string // color1 - red/error
	dim        string // color8 - dimmed text
}

var themes = map[string]themeColors{
	"catppuccin": {
		background: "#1e1e2e",
		foreground: "#cdd6f4",
		accent:     "#89b4fa",
		cursor:     "#f5e0dc",
		errColor:   "#f38ba8",
		dim:        "#585b70",
	},
	"catppuccin-latte": {
		background: "#eff1f5",
		foreground: "#4c4f69",
		accent:     "#1e66f5",
		cursor:     "#dc8a78",
		errColor:   "#d20f39",
		dim:        "#acb0be",
	},
	"ethereal": {
		background: "#060B1E",
		foreground: "#ffcead",
		accent:     "#7d82d9",
		cursor:     "#ffcead",
		errColor:   "#ED5B5A",
		dim:        "#6d7db6",
	},
	"everforest": {
		background: "#2d353b",
		foreground: "#d3c6aa",
		accent:     "#7fbbb3",
		cursor:     "#d3c6aa",
		errColor:   "#e67e80",
		dim:        "#475258",
	},
	"flexoki-light": {
		background: "#FFFCF0",
		foreground: "#100F0F",
		accent:     "#205EA6",
		cursor:     "#100F0F",
		errColor:   "#D14D41",
		dim:        "#100F0F",
	},
	"gruvbox": {
		background: "#282828",
		foreground: "#d4be98",
		accent:     "#7daea3",
		cursor:     "#bdae93",
		errColor:   "#ea6962",
		dim:        "#3c3836",
	},
	"hackerman": {
		background: "#0B0C16",
		foreground: "#ddf7ff",
		accent:     "#82FB9C",
		cursor:     "#ddf7ff",
		errColor:   "#50f872",
		dim:        "#6a6e95",
	},
	"kanagawa": {
		background: "#1f1f28",
		foreground: "#dcd7ba",
		accent:     "#7e9cd8",
		cursor:     "#c8c093",
		errColor:   "#c34043",
		dim:        "#727169",
	},
	"matte-black": {
		background: "#121212",
		foreground: "#bebebe",
		accent:     "#e68e0d",
		cursor:     "#eaeaea",
		errColor:   "#D35F5F",
		dim:        "#8a8a8d",
	},
	"nord": {
		background: "#2e3440",
		foreground: "#d8dee9",
		accent:     "#81a1c1",
		cursor:     "#d8dee9",
		errColor:   "#bf616a",
		dim:        "#4c566a",
	},
	"osaka-jade": {
		background: "#111c18",
		foreground: "#C1C497",
		accent:     "#509475",
		cursor:     "#D7C995",
		errColor:   "#FF5345",
		dim:        "#53685B",
	},
	"ristretto": {
		background: "#2c2525",
		foreground: "#e6d9db",
		accent:     "#f38d70",
		cursor:     "#c3b7b8",
		errColor:   "#fd6883",
		dim:        "#948a8b",
	},
	"rose-pine": {
		background: "#faf4ed",
		foreground: "#575279",
		accent:     "#56949f",
		cursor:     "#cecacd",
		errColor:   "#b4637a",
		dim:        "#9893a5",
	},
	"tokyo-night": {
		background: "#1a1b26",
		foreground: "#a9b1d6",
		accent:     "#7aa2f7",
		cursor:     "#c0caf5",
		errColor:   "#f7768e",
		dim:        "#444b6a",
	},
}

var themeOrder = []string{
	"catppuccin",
	"catppuccin-latte",
	"ethereal",
	"everforest",
	"flexoki-light",
	"gruvbox",
	"hackerman",
	"kanagawa",
	"matte-black",
	"nord",
	"osaka-jade",
	"ristretto",
	"rose-pine",
	"tokyo-night",
}

func hexToColor(hex string) tcell.Color {
	hex = strings.TrimPrefix(hex, "#")
	r, _ := strconv.ParseInt(hex[0:2], 16, 32)
	g, _ := strconv.ParseInt(hex[2:4], 16, 32)
	b, _ := strconv.ParseInt(hex[4:6], 16, 32)
	return tcell.NewRGBColor(int32(r), int32(g), int32(b))
}

func blendColors(c1, c2 tcell.Color, ratio float64) tcell.Color {
	r1, g1, b1 := c1.RGB()
	r2, g2, b2 := c2.RGB()
	r := int32(float64(r1)*(1-ratio) + float64(r2)*ratio)
	g := int32(float64(g1)*(1-ratio) + float64(g2)*ratio)
	b := int32(float64(b1)*(1-ratio) + float64(b2)*ratio)
	return tcell.NewRGBColor(r, g, b)
}

func detectOmarchyTheme() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "omarchy", "current", "theme.name"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func themeConfigPath() string {
	dataDir := os.Getenv("XDG_CONFIG_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".config")
	}
	return filepath.Join(dataDir, "term-type", "theme")
}

func loadThemePreference() string {
	data, err := os.ReadFile(themeConfigPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveThemePreference(name string) {
	path := themeConfigPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(name+"\n"), 0o644)
}

func clearThemePreference() error {
	path := themeConfigPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func resolveThemeName(name string) string {
	if name != "" {
		return name
	}
	if saved := loadThemePreference(); saved != "" {
		return saved
	}
	if detected := detectOmarchyTheme(); detected != "" {
		return detected
	}
	return "tokyo-night"
}

func initTheme(name string) {
	name = resolveThemeName(name)
	t, ok := themes[name]
	if !ok {
		t = themes["tokyo-night"]
	}

	bg := hexToColor(t.background)
	errC := hexToColor(t.errColor)

	colorBackground = bg
	colorCorrect = hexToColor(t.foreground)
	colorAccent = hexToColor(t.accent)
	colorCursor = hexToColor(t.cursor)
	colorPending = hexToColor(t.dim)
	colorSubtle = hexToColor(t.dim)
	colorWrongFg = errC
	colorWrongBg = blendColors(bg, errC, 0.25)
}
