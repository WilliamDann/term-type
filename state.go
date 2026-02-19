package main

import (
	"fmt"
	"time"
)

type TestState struct {
	Target    string // the full target text
	Input     []rune // what the user has typed so far
	StartTime time.Time
	EndTime   time.Time
	Started   bool
	Finished  bool

	// Mode info
	TimedMode    bool
	TimeLimitSec int
	WordCount    int
	PipedText    string // original piped text for retry
}

func NewTestState(target string, timedMode bool, timeLimitSec int, wordCount int) *TestState {
	return &TestState{
		Target:       target,
		Input:        make([]rune, 0, len(target)),
		TimedMode:    timedMode,
		TimeLimitSec: timeLimitSec,
		WordCount:    wordCount,
	}
}

func (s *TestState) HandleChar(ch rune) {
	if s.Finished {
		return
	}
	if !s.Started {
		s.Started = true
		s.StartTime = time.Now()
	}
	// Don't allow typing past the target length
	if len(s.Input) >= len([]rune(s.Target)) {
		return
	}
	s.Input = append(s.Input, ch)

	// In word mode, finish when all characters are typed
	if !s.TimedMode && len(s.Input) == len([]rune(s.Target)) {
		s.Finish()
	}
}

func (s *TestState) HandleBackspace() {
	if s.Finished || len(s.Input) == 0 {
		return
	}
	s.Input = s.Input[:len(s.Input)-1]
}

func (s *TestState) HandleDeleteWord() {
	if s.Finished || len(s.Input) == 0 {
		return
	}
	// Delete trailing spaces
	for len(s.Input) > 0 && s.Input[len(s.Input)-1] == ' ' {
		s.Input = s.Input[:len(s.Input)-1]
	}
	// Delete until space or empty
	for len(s.Input) > 0 && s.Input[len(s.Input)-1] != ' ' {
		s.Input = s.Input[:len(s.Input)-1]
	}
}

func (s *TestState) Finish() {
	if !s.Finished {
		s.Finished = true
		s.EndTime = time.Now()
	}
}

func (s *TestState) Elapsed() time.Duration {
	if !s.Started {
		return 0
	}
	if s.Finished {
		return s.EndTime.Sub(s.StartTime)
	}
	return time.Since(s.StartTime)
}

func (s *TestState) TimeRemaining() float64 {
	if !s.TimedMode || !s.Started {
		return float64(s.TimeLimitSec)
	}
	rem := float64(s.TimeLimitSec) - s.Elapsed().Seconds()
	if rem < 0 {
		return 0
	}
	return rem
}

func (s *TestState) CorrectChars() int {
	target := []rune(s.Target)
	count := 0
	for i, ch := range s.Input {
		if i < len(target) && ch == target[i] {
			count++
		}
	}
	return count
}

func (s *TestState) WrongChars() int {
	target := []rune(s.Target)
	count := 0
	for i, ch := range s.Input {
		if i < len(target) && ch != target[i] {
			count++
		}
	}
	return count
}

func (s *TestState) WPM() float64 {
	elapsed := s.Elapsed().Minutes()
	if elapsed == 0 {
		return 0
	}
	return (float64(s.CorrectChars()) / 5.0) / elapsed
}

func (s *TestState) Accuracy() float64 {
	total := len(s.Input)
	if total == 0 {
		return 100
	}
	return float64(s.CorrectChars()) / float64(total) * 100
}

func (s *TestState) ModeString() string {
	if s.PipedText != "" {
		return fmt.Sprintf("pipe (%d words)", s.WordCount)
	}
	if s.TimedMode {
		return fmt.Sprintf("%ds", s.TimeLimitSec)
	}
	return fmt.Sprintf("%d words", s.WordCount)
}
