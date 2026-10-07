// Copyright (c) 2026 André Gonçalves. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestWrapText(t *testing.T) {
	for _, test := range []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{name: "words", text: "Computation can be delegated without delegating the decision.", width: 24,
			want: []string{"Computation can be", "delegated without", "delegating the decision."}},
		{name: "paragraphs", text: "First line.\r\n\r\nSecond line.", width: 20,
			want: []string{"First line.", "", "Second line."}},
		{name: "long word", text: "See abcdefghijkl next", width: 5,
			want: []string{"See", "abcde", "fghij", "kl", "next"}},
		{name: "wide characters", text: "地球 火星 土星", width: 9,
			want: []string{"地球 火星", "土星"}},
		{name: "wide word", text: "地球火星土星", width: 5,
			want: []string{"地球", "火星", "土星"}},
		{name: "combining characters", text: "cafe\u0301 cafe\u0301", width: 4,
			want: []string{"cafe\u0301", "cafe\u0301"}},
		{name: "empty", width: 10, want: []string{""}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := wrapText(test.text, test.width); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("wrapped lines = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestPrintBox(t *testing.T) {
	var output bytes.Buffer
	printBox(&widthOutput{Writer: &output, width: 24}, "Question rationale", "Earth bears life.\n\nNo life on Mars.")
	want := `
+----------------------+
| Question rationale   |
+----------------------+
| Earth bears life.    |
|                      |
| No life on Mars.     |
+----------------------+
`
	if got := output.String(); got != want {
		t.Fatalf("box output:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintQuestionLayout(t *testing.T) {
	for _, test := range []struct {
		name   string
		second bool
		prompt string
	}{
		{name: "single", prompt: "What planet bears life? (Select 1 answer.)"},
		{name: "multiple", second: true, prompt: "What planet bears life? (Select 2 answers.)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			question := Question{Question: "What planet bears life?", Options: []Option{
				{Option: "Earth", Correct: true},
				{Option: "Mars", Correct: test.second},
			}}
			var output bytes.Buffer
			printQuestion(&output, "Question 3 of 60", question)
			border := "+" + strings.Repeat("-", 78) + "+"
			want := []string{border, "Question 3 of 60", border, test.prompt, border,
				"Option 1 - Earth", "Option 2 - Mars", border}
			if got := boxContents(output.String()); !reflect.DeepEqual(got, want) {
				t.Fatalf("question layout = %#v, want %#v", got, want)
			}
		})
	}
}

func TestPrintRationaleLayout(t *testing.T) {
	question := Question{Rationale: "Only Earth is known to bear life.", Options: []Option{
		{Option: "Earth", Correct: true, Rationale: "Earth bears life."},
		{Option: "Mars", Rationale: "No confirmed life found."},
		{Option: "Another answer", Correct: true, Rationale: "Another explanation."},
	}}
	var output bytes.Buffer
	printRationale(&output, question, []int{1, 2})
	border := "+" + strings.Repeat("-", 78) + "+"
	want := []string{
		border, "Incorrect.", "", "Only Earth is known to bear life.", border,
		"Option 1 (correct)", "Earth", "", "Correct.", "Earth bears life.", border,
		"Option 2 (selected)", "Mars", "", "Incorrect.", "No confirmed life found.", border,
		"Option 3 (correct, selected)", "Another answer", "", "Correct.", "Another explanation.", border,
	}
	if got := boxContents(output.String()); !reflect.DeepEqual(got, want) {
		t.Fatalf("feedback layout = %#v, want %#v", got, want)
	}
}

func boxContents(text string) []string {
	lines := strings.Split(strings.Trim(text, "\n"), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "| ") && strings.HasSuffix(line, " |") {
			lines[i] = strings.TrimSpace(line[2 : len(line)-2])
		}
	}
	return lines
}

func TestPrintBoxFitsWidth(t *testing.T) {
	for _, text := range []string{
		"Computation can be delegated without delegating the decision based on it.",
		"source: https://example.com/a/very/long/unbroken/reference",
		"地球 bears life. Mars does not.\n\nRésumé: cafe\u0301.",
	} {
		var output bytes.Buffer
		printBox(&widthOutput{Writer: &output, width: 20}, "Option 2 (correct, selected)", text)
		for _, line := range strings.Split(strings.Trim(output.String(), "\n"), "\n") {
			if width := runewidth.StringWidth(line); width != 20 {
				t.Fatalf("line width = %d, want 20: %q", width, line)
			}
		}
	}
}

func TestBoxWidth(t *testing.T) {
	for _, test := range []struct {
		width int
		want  int
	}{
		{width: 0, want: 80},
		{width: 40, want: 40},
		{width: 120, want: 120},
		{width: 8, want: 8},
	} {
		var output bytes.Buffer
		if got := boxWidth(&widthOutput{Writer: &output, width: test.width}); got != test.want {
			t.Fatalf("configured width %d = %d, want %d", test.width, got, test.want)
		}
	}
}
