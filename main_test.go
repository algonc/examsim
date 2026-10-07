// Copyright (c) 2026 André Gonçalves. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const (
	testSessionID       = "0bd73aa1-af51-45cd-af81-544d65239a4b"
	secondTestSessionID = "1bd73aa1-af51-45cd-bf81-544d65239a4b"
)

func TestRunHelp(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"-help"}, strings.NewReader(""), &output); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := output.String()
	for _, want := range []string{
		"examsim - Exam Simulator",
		"Usage:",
		"Options:",
		"-e <path>",
		"-i, --instant-feedback",
		"-resume <session-id>",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("help output missing %q:\n%s", want, text)
		}
	}
}

func TestRunCompletesExamWithDefaultSessionStore(t *testing.T) {
	temporaryHome := t.TempDir()
	t.Setenv("HOME", temporaryHome)
	t.Setenv("USERPROFILE", temporaryHome)

	examPath := filepath.Join(t.TempDir(), "exam.yaml")
	if err := os.WriteFile(examPath, []byte(`
name: Run exam
questions:
  - question: Question
    options:
      - option: Answer
        correct: true
        rationale: Correct.
`), 0600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := run([]string{"-e", examPath}, strings.NewReader("1\n"), &output); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(output.String(), "Result: 100% (1 correct of 1 questions).") {
		t.Fatalf("unexpected output:\n%s", output.String())
	}

	entries, err := os.ReadDir(filepath.Join(temporaryHome, ".examsim", "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("completed session files were not removed: %v", entries)
	}
}

func TestParseExamYAML(t *testing.T) {
	exam, err := parseExamYAML([]byte(`
name: "Exam 1"
questions:
- question: What planet bears life? # inline comments are ignored
  options:
  - option: Sun
    correct: false
    rationale: Not even a planet.
  - option: Earth
    correct: true
    rationale: Earth bears life.
  rationale: Only Earth is known to bear life.
- question: What planet has rings?
  options:
  - option: Saturn
    correct: true
    rationale: Saturn has rings.
  - option: Uranus
    correct: true
    rationale: Uranus has rings too.
`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exam.Name != "Exam 1" {
		t.Fatalf("name = %q, want Exam 1", exam.Name)
	}
	if len(exam.Questions) != 2 {
		t.Fatalf("questions = %d, want 2", len(exam.Questions))
	}
	if got := exam.Questions[0].Options[1]; got.Option != "Earth" || !got.Correct || got.Rationale != "Earth bears life." {
		t.Fatalf("unexpected parsed option: %#v", got)
	}
	if got := exam.Questions[0].Rationale; got != "Only Earth is known to bear life." {
		t.Fatalf("question rationale = %q", got)
	}
	if got := exam.Questions[1].Rationale; got != "" {
		t.Fatalf("omitted question rationale = %q, want empty", got)
	}
	if countCorrect(exam.Questions[1]) != 2 {
		t.Fatalf("second question correct count = %d, want 2", countCorrect(exam.Questions[1]))
	}
}

func TestParseExamYAMLSupportsBlockScalars(t *testing.T) {
	exam, err := parseExamYAML([]byte(`
name: Block scalar exam
questions:
  - question: >
      What planet
      bears life?
    options:
      - option: Earth
        correct: true
        rationale: |
          Earth is the only planet known
          to bear life.
      - option: Mars
        correct: false
        rationale: No confirmed life has been found.
    rationale: >-
      Only Earth is known to bear life.
      No confirmed life has been found on Mars.
`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(exam.Questions[0].Question); got != "What planet bears life?" {
		t.Fatalf("question = %q", got)
	}
	if got := exam.Questions[0].Options[0].Rationale; !strings.Contains(got, "known\nto bear life") {
		t.Fatalf("rationale did not preserve the literal block: %q", got)
	}
	if got := exam.Questions[0].Rationale; got != "Only Earth is known to bear life. No confirmed life has been found on Mars." {
		t.Fatalf("question rationale did not fold the block: %q", got)
	}
}

func TestParseExamYAMLQuestionRationale(t *testing.T) {
	for _, test := range []struct {
		name  string
		field string
		want  string
	}{
		{name: "omitted"},
		{name: "empty", field: `    rationale: ""`},
		{name: "blank", field: `    rationale: "  "`, want: "  "},
		{name: "literal", field: "    rationale: |-\n      First line.\n      Second line.", want: "First line.\nSecond line."},
	} {
		t.Run(test.name, func(t *testing.T) {
			exam, err := parseExamYAML([]byte(`name: Exam
questions:
  - question: Q
    options:
      - option: A
        correct: true
        rationale: Correct.
` + test.field + "\n"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := exam.Questions[0].Rationale; got != test.want {
				t.Fatalf("question rationale = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseExamYAMLRejectsInvalidSchema(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "unknown field",
			yaml: `
name: Exam
questions:
  - question: Q
    options:
      - option: A
        correct: true
        rationle: Typo
`,
			want: "field rationle not found",
		},
		{
			name: "missing correct",
			yaml: `
name: Exam
questions:
  - question: Q
    options:
      - option: A
        rationale: Required
`,
			want: `field "correct" is required`,
		},
		{
			name: "empty rationale",
			yaml: `
name: Exam
questions:
  - question: Q
    options:
      - option: A
        correct: true
        rationale: ""
`,
			want: "empty rationale",
		},
		{
			name: "duplicate option",
			yaml: `
name: Exam
questions:
  - question: Q
    options:
      - option: A
        correct: true
        rationale: First
      - option: A
        correct: false
        rationale: Second
`,
			want: "duplicate option",
		},
		{
			name: "multiple documents",
			yaml: `
name: Exam
questions: []
---
name: Another exam
questions: []
`,
			want: "exactly one document",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseExamYAML([]byte(test.yaml))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestParseCLIRejectsIgnoredArgumentsAndResumeOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "unexpected positional argument",
			args: []string{"-e", "exam.yaml", "extra", "-q", "2"},
			want: "unexpected argument(s)",
		},
		{
			name: "question limit on resume",
			args: []string{"-resume", testSessionID, "-q", "2"},
			want: "can only be used when starting",
		},
		{
			name: "instant feedback on resume",
			args: []string{"-resume", testSessionID, "--instant-feedback"},
			want: "can only be used when starting",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := parseCLI(test.args, io.Discard)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestParseAnswer(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		optionCount int
		required    int
		want        []int
		wantErr     bool
	}{
		{
			name:        "single answer",
			input:       "3\n",
			optionCount: 3,
			required:    1,
			want:        []int{2},
		},
		{
			name:        "multiple answers sorted",
			input:       "3, 1",
			optionCount: 3,
			required:    2,
			want:        []int{0, 2},
		},
		{
			name:        "requires exact answer count",
			input:       "1",
			optionCount: 3,
			required:    2,
			wantErr:     true,
		},
		{
			name:        "rejects duplicates",
			input:       "1 1",
			optionCount: 3,
			required:    2,
			wantErr:     true,
		},
		{
			name:        "rejects out of range",
			input:       "4",
			optionCount: 3,
			required:    1,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAnswer(tt.input, tt.optionCount, tt.required)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestAnswerCorrect(t *testing.T) {
	question := Question{
		Question: "What planet has rings?",
		Options: []Option{
			{Option: "Uranus", Correct: true},
			{Option: "Saturn", Correct: true},
			{Option: "Sun", Correct: false},
		},
	}

	if !answerCorrect(question, []int{0, 1}) {
		t.Fatal("expected both correct options to pass")
	}
	if answerCorrect(question, []int{0}) {
		t.Fatal("expected missing correct option to fail")
	}
	if answerCorrect(question, []int{0, 2}) {
		t.Fatal("expected selecting an incorrect option to fail")
	}
}

func TestScoreSession(t *testing.T) {
	session := &Session{
		Questions: []Question{
			{
				Question: "Q1",
				Options: []Option{
					{Option: "A", Correct: true},
					{Option: "B", Correct: false},
				},
			},
			{
				Question: "Q2",
				Options: []Option{
					{Option: "A", Correct: false},
					{Option: "B", Correct: true},
				},
			},
		},
		Answers: [][]int{{0}, {0}},
	}

	got := scoreSession(session)
	if got.correct != 1 {
		t.Fatalf("correct = %d, want 1", got.correct)
	}
	if len(got.wrongItems) != 1 {
		t.Fatalf("wrongItems = %d, want 1", len(got.wrongItems))
	}
	if got.wrongItems[0].number != 2 {
		t.Fatalf("wrong item number = %d, want 2", got.wrongItems[0].number)
	}
}

func TestSessionStoreRejectsTraversal(t *testing.T) {
	store := &sessionStore{dir: t.TempDir()}
	for _, id := range []string{"../../outside", `..\..\outside`, "not-a-uuid"} {
		if _, err := store.path(id); err == nil {
			t.Fatalf("expected session ID %q to be rejected", id)
		}
	}
}

func TestSessionStoreRoundTripAndAtomicOverwrite(t *testing.T) {
	store := &sessionStore{dir: t.TempDir()}
	session := validTestSession()
	session.Questions[0].Rationale = "Whole question explanation.\nAnother line."
	session.Current = 1
	session.Answers[0] = []int{0}
	if err := store.save(session); err != nil {
		t.Fatalf("first save: %v", err)
	}

	session.Current = 2
	session.Answers[1] = []int{0}
	if err := store.save(session); err != nil {
		t.Fatalf("replacement save: %v", err)
	}

	loaded, err := store.load(session.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Current != 2 || len(loaded.Answers[1]) != 1 || loaded.Answers[1][0] != 0 {
		t.Fatalf("unexpected loaded progress: %#v", loaded)
	}
	if loaded.Questions[0].Rationale != session.Questions[0].Rationale || loaded.Questions[1].Rationale != "" {
		t.Fatalf("unexpected loaded question rationales: %#v", loaded.Questions)
	}

	entries, err := os.ReadDir(store.dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != session.ID+".json" {
		t.Fatalf("unexpected session directory contents: %v", entries)
	}
}

func TestConductSessionQuestionRationale(t *testing.T) {
	const rationale = "Only Earth is known to bear life.\nNo confirmed life on Mars."
	for _, test := range []struct {
		name            string
		instantFeedback bool
		answer          string
		rationale       string
		wantCount       int
	}{
		{name: "summary", answer: "2\n", rationale: rationale, wantCount: 1},
		{name: "instant and summary", instantFeedback: true, answer: "2\n", rationale: rationale, wantCount: 2},
		{name: "correct answer", instantFeedback: true, answer: "1\n", rationale: rationale},
		{name: "omitted", instantFeedback: true, answer: "2\n"},
		{name: "blank", instantFeedback: true, answer: "2\n", rationale: " \n\t"},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := validTestSession()
			session.InstantFeedback = test.instantFeedback
			session.Questions = []Question{{
				Question:  "What planet bears life?",
				Rationale: test.rationale,
				Options: []Option{
					{Option: "Earth", Correct: true, Rationale: "Earth bears life."},
					{Option: "Mars", Rationale: "No confirmed life found."},
				},
			}}
			session.Answers = make([][]int, 1)
			store := &sessionStore{dir: t.TempDir()}
			var output bytes.Buffer
			if err := conductSession(session, strings.NewReader(test.answer), &output, store, nil); err != nil {
				t.Fatalf("conduct session: %v", err)
			}
			text := output.String()
			if got := strings.Count(text, "Rationale: "); got != test.wantCount {
				t.Fatalf("question rationale count = %d, want %d:\n%s", got, test.wantCount, text)
			}
			if test.wantCount > 0 {
				want := "Rationale: " + rationale + "\n- Earth (correct): Earth bears life.\n- Mars (selected): No confirmed life found.\n"
				if got := strings.Count(text, want); got != test.wantCount {
					t.Fatalf("missing question or option feedback:\n%s", text)
				}
				if strings.Index(text, "Rationale: ") < strings.Index(text, "Answer: ") {
					t.Fatalf("question rationale was revealed before answering:\n%s", text)
				}
			} else if test.answer == "2\n" && !strings.Contains(text, "- Mars (selected): No confirmed life found.") {
				t.Fatalf("missing option feedback:\n%s", text)
			}
		})
	}
}

func TestSessionStoreRejectsMismatchedEmbeddedID(t *testing.T) {
	store := &sessionStore{dir: t.TempDir()}
	session := validTestSession()
	session.ID = secondTestSessionID
	content, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	path, err := store.path(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = store.load(testSessionID)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("error = %v, want mismatched ID error", err)
	}
}

func TestValidateSessionRejectsInvalidSavedAnswers(t *testing.T) {
	session := validTestSession()
	session.Current = 1
	session.Answers[0] = []int{4}
	if err := validateSession(session); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("error = %v, want out-of-range error", err)
	}

	session = validTestSession()
	session.Answers[1] = []int{0}
	if err := validateSession(session); err == nil || !strings.Contains(err.Error(), "unanswered") {
		t.Fatalf("error = %v, want unanswered-selection error", err)
	}
}

func TestExecuteInterruptsAndResumesSession(t *testing.T) {
	examPath := filepath.Join(t.TempDir(), "exam.yaml")
	if err := os.WriteFile(examPath, []byte(`
name: Lifecycle exam
questions:
  - question: First
    options:
      - option: Answer
        correct: true
        rationale: Correct.
  - question: Second
    options:
      - option: Answer
        correct: true
        rationale: Correct.
`), 0600); err != nil {
		t.Fatal(err)
	}

	store := &sessionStore{dir: t.TempDir()}
	interrupts := make(chan os.Signal, 1)
	release := make(chan struct{})
	input := &signalAfterFirstLineReader{
		first:   strings.NewReader("1\n"),
		signal:  interrupts,
		release: release,
	}
	var interruptedOutput bytes.Buffer
	if err := execute(cliConfig{examPath: examPath}, input, &interruptedOutput, store, interrupts); err != nil {
		close(release)
		t.Fatalf("interrupted execution: %v", err)
	}
	close(release)
	if !strings.Contains(interruptedOutput.String(), "Resume this session with") {
		t.Fatalf("missing resume instructions:\n%s", interruptedOutput.String())
	}

	entries, err := os.ReadDir(store.dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("session files = %d, want 1", len(entries))
	}
	resumeID := strings.TrimSuffix(entries[0].Name(), ".json")
	saved, err := store.load(resumeID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Current != 1 {
		t.Fatalf("saved progress = %d, want 1", saved.Current)
	}

	var resumedOutput bytes.Buffer
	if err := execute(cliConfig{resumeID: resumeID}, strings.NewReader("1\n"), &resumedOutput, store, nil); err != nil {
		t.Fatalf("resumed execution: %v", err)
	}
	if !strings.Contains(resumedOutput.String(), "Result: 100% (2 correct of 2 questions).") {
		t.Fatalf("unexpected resumed output:\n%s", resumedOutput.String())
	}
	if _, err := os.Stat(filepath.Join(store.dir, resumeID+".json")); !os.IsNotExist(err) {
		t.Fatalf("completed session was not removed: %v", err)
	}
}

type signalAfterFirstLineReader struct {
	first    *strings.Reader
	signal   chan<- os.Signal
	release  <-chan struct{}
	signaled bool
}

func (reader *signalAfterFirstLineReader) Read(buffer []byte) (int, error) {
	if reader.first.Len() > 0 {
		return reader.first.Read(buffer)
	}
	if !reader.signaled {
		reader.signaled = true
		reader.signal <- syscall.SIGTERM
	}
	<-reader.release
	return 0, io.EOF
}

func validTestSession() *Session {
	return &Session{
		ID:       testSessionID,
		ExamName: "Test exam",
		ExamPath: "exam.yaml",
		Questions: []Question{
			{
				Question: "Question 1",
				Options:  []Option{{Option: "Answer", Correct: true, Rationale: "Correct."}},
			},
			{
				Question: "Question 2",
				Options:  []Option{{Option: "Answer", Correct: true, Rationale: "Correct."}},
			},
		},
		Answers: make([][]int, 2),
	}
}
