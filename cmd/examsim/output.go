// Copyright (c) 2026 André Gonçalves. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

const defaultBoxWidth = 80

type widthOutput struct {
	io.Writer
	width int
}

func boxWidth(output io.Writer) int {
	if configured, ok := output.(*widthOutput); ok {
		if configured.width > 0 {
			return configured.width
		}
		return boxWidth(configured.Writer)
	}
	width := defaultBoxWidth
	if file, ok := output.(*os.File); ok {
		if columns, _, err := term.GetSize(int(file.Fd())); err == nil && columns >= 8 {
			return min(width, columns)
		}
	}
	return width
}

func printBox(output io.Writer, sections ...string) {
	width := boxWidth(output) - 4
	border := strings.Repeat("─", width+2)
	fmt.Fprintln(output, "\n┌"+border+"┐")
	for i, section := range sections {
		if i > 0 {
			fmt.Fprintln(output, "├"+border+"┤")
		}
		printBoxLines(output, section, width)
	}
	fmt.Fprintln(output, "└"+border+"┘")
}

func printQuestion(output io.Writer, title string, question Question) {
	required := countCorrect(question)
	choices := "answer"
	if required != 1 {
		choices = "answers"
	}
	prompt := fmt.Sprintf("%s (Select %d %s.)", question.Question, required, choices)
	width := boxWidth(output) - 4
	border := strings.Repeat("─", width+2)
	fmt.Fprintln(output, "\n┌"+border+"┐")
	printBoxLines(output, title, width)
	fmt.Fprintln(output, "├"+border+"┤")
	printBoxLines(output, prompt, width)
	printBoxLine(output, "", width)
	for i, option := range question.Options {
		prefix := fmt.Sprintf("%d - ", i+1)
		if width-len(prefix) < 2 {
			printBoxLines(output, prefix+option.Option, width)
		} else {
			for j, line := range wrapText(option.Option, width-len(prefix)) {
				if j == 0 {
					printBoxLine(output, prefix+line, width)
				} else {
					printBoxLine(output, strings.Repeat(" ", len(prefix))+line, width)
				}
			}
		}
		printBoxLine(output, "", width)
	}
	fmt.Fprintln(output, "└"+border+"┘")
}

func printBoxLines(output io.Writer, text string, width int) {
	for _, line := range wrapText(text, width) {
		printBoxLine(output, line, width)
	}
}

func printBoxLine(output io.Writer, line string, width int) {
	padding := strings.Repeat(" ", width-runewidth.StringWidth(line))
	fmt.Fprintf(output, "│ %s%s │\n", line, padding)
}

func wrapText(text string, width int) []string {
	var wrapped []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := ""
		for _, word := range strings.Fields(paragraph) {
			if line != "" && runewidth.StringWidth(line)+1+runewidth.StringWidth(word) > width {
				wrapped = append(wrapped, line)
				line = ""
			}
			if runewidth.StringWidth(word) > width {
				parts := strings.Split(runewidth.Wrap(word, width), "\n")
				wrapped = append(wrapped, parts[:len(parts)-1]...)
				line = parts[len(parts)-1]
			} else if line == "" {
				line = word
			} else {
				line += " " + word
			}
		}
		wrapped = append(wrapped, line)
	}
	return wrapped
}
