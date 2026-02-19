package main

import (
	"embed"
	"math/rand"
	"strings"
)

//go:embed words.txt
var wordsFile embed.FS

var wordList []string

func init() {
	data, _ := wordsFile.ReadFile("words.txt")
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		w := strings.TrimSpace(line)
		if w != "" {
			wordList = append(wordList, w)
		}
	}
}

func pickWords(n int) string {
	words := make([]string, n)
	for i := range words {
		words[i] = wordList[rand.Intn(len(wordList))]
	}
	return strings.Join(words, " ")
}
