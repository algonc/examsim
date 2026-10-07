// Copyright (c) 2026 André Gonçalves. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

const appName = "examsim"

var errInterrupted = errors.New("exam interrupted")

type multiFlag struct {
	value bool
}

func (f *multiFlag) String() string {
	return strconv.FormatBool(f.value)
}

func (f *multiFlag) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	f.value = parsed
	return nil
}

func (f *multiFlag) IsBoolFlag() bool {
	return true
}

type cliConfig struct {
	examPath        string
	resumeID        string
	questionLimit   int
	outputWidth     int
	instantFeedback bool
}

type result struct {
	correct    int
	wrongItems []wrongItem
}

type wrongItem struct {
	number   int
	question Question
	selected []int
}

type lineResult struct {
	line string
	err  error
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	config, proceed, err := parseCLI(args, output)
	if err != nil || !proceed {
		return err
	}

	store, err := newSessionStore()
	if err != nil {
		return err
	}

	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(interrupted)

	return execute(config, input, output, store, interrupted)
}

func parseCLI(args []string, output io.Writer) (cliConfig, bool, error) {
	flags := flag.NewFlagSet(appName, flag.ContinueOnError)
	flags.SetOutput(output)
	flags.Usage = func() {
		printHelp(output)
	}

	var config cliConfig
	instantFeedback := &multiFlag{}

	flags.StringVar(&config.examPath, "e", "", "exam YAML file")
	flags.StringVar(&config.resumeID, "resume", "", "resume a saved session ID")
	flags.IntVar(&config.questionLimit, "q", 0, "number of random questions to include")
	flags.IntVar(&config.outputWidth, "output-width", 0, "box width in columns (0 for automatic sizing)")
	flags.Var(instantFeedback, "i", "show feedback after each question")
	flags.Var(instantFeedback, "instant-feedback", "show feedback after each question")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cliConfig{}, false, nil
		}
		return cliConfig{}, false, err
	}
	if flags.NArg() > 0 {
		return cliConfig{}, false, fmt.Errorf("unexpected argument(s): %s", strings.Join(flags.Args(), " "))
	}

	config.instantFeedback = instantFeedback.value
	if config.examPath == "" && config.resumeID == "" {
		return cliConfig{}, false, errors.New("provide -e exam.yaml to start, or -resume session-id to continue")
	}
	if config.examPath != "" && config.resumeID != "" {
		return cliConfig{}, false, errors.New("use either -e or -resume, not both")
	}
	if config.questionLimit < 0 {
		return cliConfig{}, false, errors.New("-q must be zero or greater")
	}
	if config.outputWidth != 0 && config.outputWidth < 8 {
		return cliConfig{}, false, errors.New("-output-width must be zero (automatic) or at least 8 columns")
	}

	setFlags := map[string]bool{}
	flags.Visit(func(current *flag.Flag) {
		setFlags[current.Name] = true
	})
	if config.resumeID != "" && (setFlags["q"] || setFlags["i"] || setFlags["instant-feedback"]) {
		return cliConfig{}, false, errors.New("-q and --instant-feedback can only be used when starting a new exam")
	}

	return config, true, nil
}

func execute(config cliConfig, input io.Reader, output io.Writer, store *sessionStore, interrupted <-chan os.Signal) error {
	output = &widthOutput{Writer: output, width: config.outputWidth}
	session, err := loadOrCreateSession(store, config)
	if err != nil {
		return err
	}

	// Persist the randomized exam before prompting so even an immediate
	// interruption has a complete, resumable snapshot.
	if err := store.save(session); err != nil {
		return err
	}

	err = conductSession(session, input, output, store, interrupted)
	if errors.Is(err, errInterrupted) || (errors.Is(err, io.EOF) && session.Current < len(session.Questions)) {
		if saveErr := store.save(session); saveErr != nil {
			return saveErr
		}
		_, writeErr := fmt.Fprintf(output, "\nExiting. Resume this session with:\n\n%s -resume %s\n", appName, session.ID)
		return writeErr
	}
	if err != nil {
		return err
	}
	return store.remove(session.ID)
}

func printHelp(output io.Writer) {
	fmt.Fprintf(output, `%s - Exam Simulator

Usage:
  %s -e <exam.yaml> [options]
  %s -resume <session-id> [-output-width <columns>]
  %s -help

Options:
  -e <path>                 Load an exam YAML file and start a new session.
  -q <count>                Use a random subset when starting a new exam.
  -output-width <columns>   Set box width; 0 uses automatic sizing (default).
  -i, --instant-feedback    Show correctness and wrong-answer rationales immediately.
  -resume <session-id>      Resume a saved session.
  -h, -help, --help         Show this help message.
`, appName, appName, appName, appName)
}

func loadOrCreateSession(store *sessionStore, config cliConfig) (*Session, error) {
	if config.resumeID != "" {
		return store.load(config.resumeID)
	}

	exam, err := loadExam(config.examPath)
	if err != nil {
		return nil, err
	}
	if config.questionLimit > len(exam.Questions) {
		return nil, fmt.Errorf("-q %d exceeds available questions (%d)", config.questionLimit, len(exam.Questions))
	}

	questions := cloneQuestions(exam.Questions)
	if err := shuffleQuestions(questions); err != nil {
		return nil, err
	}
	if config.questionLimit > 0 {
		questions = questions[:config.questionLimit]
	}
	for i := range questions {
		if err := shuffleOptions(questions[i].Options); err != nil {
			return nil, err
		}
	}

	id, err := newSessionID()
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:              id,
		ExamName:        exam.Name,
		ExamPath:        config.examPath,
		InstantFeedback: config.instantFeedback,
		Questions:       questions,
		Answers:         make([][]int, len(questions)),
	}, nil
}

func conductSession(session *Session, input io.Reader, output io.Writer, store *sessionStore, interrupted <-chan os.Signal) error {
	reader := bufio.NewReader(input)
	total := len(session.Questions)

	for session.Current < total {
		question := session.Questions[session.Current]
		required := countCorrect(question)

		printQuestion(output, fmt.Sprintf("Question %d of %d", session.Current+1, total), question)

		selected, err := promptForAnswer(reader, output, len(question.Options), required, interrupted)
		if err != nil {
			return err
		}
		session.Answers[session.Current] = selected

		if session.InstantFeedback {
			if answerCorrect(question, selected) {
				printBox(output, "Correct!")
			} else {
				printRationale(output, question, selected)
			}
		}

		session.Current++
		if err := store.save(session); err != nil {
			return err
		}
	}

	printSummary(output, session)
	return nil
}

func promptForAnswer(reader *bufio.Reader, output io.Writer, optionCount, required int, interrupted <-chan os.Signal) ([]int, error) {
	for {
		if required == 1 {
			fmt.Fprint(output, "\nAnswer: ")
		} else {
			fmt.Fprintf(output, "\nAnswer (%d numbers): ", required)
		}

		read := make(chan lineResult, 1)
		go func() {
			line, err := reader.ReadString('\n')
			read <- lineResult{line: line, err: err}
		}()

		select {
		case <-interrupted:
			return nil, errInterrupted
		case result := <-read:
			if result.err != nil {
				if errors.Is(result.err, io.EOF) && strings.TrimSpace(result.line) != "" {
					return parseAnswer(result.line, optionCount, required)
				}
				return nil, result.err
			}

			selected, err := parseAnswer(result.line, optionCount, required)
			if err == nil {
				return selected, nil
			}
			fmt.Fprintf(output, "Invalid answer: %v\n", err)
		}
	}
}

func parseAnswer(input string, optionCount, required int) ([]int, error) {
	fields := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\r' || r == '\n'
	})
	if len(fields) != required {
		return nil, fmt.Errorf("select exactly %d option(s)", required)
	}

	seen := map[int]bool{}
	selected := make([]int, 0, required)
	for _, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", field)
		}
		if value < 1 || value > optionCount {
			return nil, fmt.Errorf("%d is outside the range 1-%d", value, optionCount)
		}
		index := value - 1
		if seen[index] {
			return nil, fmt.Errorf("%d was selected more than once", value)
		}
		seen[index] = true
		selected = append(selected, index)
	}
	sort.Ints(selected)
	return selected, nil
}

func printSummary(output io.Writer, session *Session) {
	result := scoreSession(session)
	total := len(session.Questions)
	percent := 0
	if total > 0 {
		percent = result.correct * 100 / total
	}

	printBox(output, "Result", fmt.Sprintf("Result: %d%% (%d correct of %d questions).", percent, result.correct, total))
	if len(result.wrongItems) == 0 {
		return
	}

	fmt.Fprintln(output, "\nRationale for incorrect answers:")
	for _, item := range result.wrongItems {
		printQuestion(output, fmt.Sprintf("Question %d of %d", item.number, total), item.question)
		printRationale(output, item.question, item.selected)
	}
}

func printRationale(output io.Writer, question Question, selected []int) {
	feedback := "Incorrect."
	if strings.TrimSpace(question.Rationale) != "" {
		feedback += "\n\n" + question.Rationale
	}
	sections := []string{feedback}
	selectedSet := indexSet(selected)
	for i, option := range question.Options {
		markers := []string{}
		if option.Correct {
			markers = append(markers, "correct")
		}
		if selectedSet[i] {
			markers = append(markers, "selected")
		}

		label := fmt.Sprintf("Option %d", i+1)
		if len(markers) > 0 {
			label = fmt.Sprintf("%s (%s)", label, strings.Join(markers, ", "))
		}
		status := "Incorrect."
		if option.Correct {
			status = "Correct."
		}
		sections = append(sections, label+"\n"+option.Option+"\n\n"+status+"\n"+option.Rationale)
	}
	printBox(output, sections...)
}

func scoreSession(session *Session) result {
	scored := result{}
	for i, question := range session.Questions {
		selected := session.Answers[i]
		if answerCorrect(question, selected) {
			scored.correct++
			continue
		}
		scored.wrongItems = append(scored.wrongItems, wrongItem{
			number:   i + 1,
			question: question,
			selected: selected,
		})
	}
	return scored
}

func answerCorrect(question Question, selected []int) bool {
	selectedSet := indexSet(selected)
	for i, option := range question.Options {
		if option.Correct != selectedSet[i] {
			return false
		}
	}
	return true
}

func indexSet(values []int) map[int]bool {
	set := map[int]bool{}
	for _, value := range values {
		set[value] = true
	}
	return set
}
