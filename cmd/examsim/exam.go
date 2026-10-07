// Copyright (c) 2026 André Gonçalves. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Exam struct {
	Name      string     `json:"name"`
	Questions []Question `json:"questions"`
}

type Question struct {
	Question  string   `json:"question"`
	Options   []Option `json:"options"`
	Rationale string   `json:"rationale,omitempty"`
}

type Option struct {
	Option    string `json:"option"`
	Correct   bool   `json:"correct"`
	Rationale string `json:"rationale"`
}

type examYAML struct {
	Name      *string         `yaml:"name"`
	Questions *[]questionYAML `yaml:"questions"`
}

type questionYAML struct {
	Question  *string       `yaml:"question"`
	Options   *[]optionYAML `yaml:"options"`
	Rationale *string       `yaml:"rationale"`
}

type optionYAML struct {
	Option    *string `yaml:"option"`
	Correct   *bool   `yaml:"correct"`
	Rationale *string `yaml:"rationale"`
}

func loadExam(path string) (*Exam, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	exam, err := parseExamYAML(content)
	if err != nil {
		return nil, err
	}
	return &exam, nil
}

func parseExamYAML(content []byte) (Exam, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)

	var document examYAML
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return Exam{}, errors.New("exam YAML is empty")
		}
		return Exam{}, fmt.Errorf("parse exam YAML: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return Exam{}, fmt.Errorf("parse exam YAML: %w", err)
		}
		return Exam{}, errors.New("exam YAML must contain exactly one document")
	}

	exam, err := document.toExam()
	if err != nil {
		return Exam{}, err
	}
	if err := validateExam(exam.Name, exam.Questions); err != nil {
		return Exam{}, err
	}
	return exam, nil
}

func (document examYAML) toExam() (Exam, error) {
	if document.Name == nil {
		return Exam{}, errors.New(`exam field "name" is required`)
	}
	if document.Questions == nil {
		return Exam{}, errors.New(`exam field "questions" is required`)
	}

	exam := Exam{Name: *document.Name, Questions: make([]Question, len(*document.Questions))}
	for i, sourceQuestion := range *document.Questions {
		if sourceQuestion.Question == nil {
			return Exam{}, fmt.Errorf(`question %d field "question" is required`, i+1)
		}
		if sourceQuestion.Options == nil {
			return Exam{}, fmt.Errorf(`question %d field "options" is required`, i+1)
		}

		question := Question{Question: *sourceQuestion.Question, Options: make([]Option, len(*sourceQuestion.Options))}
		if sourceQuestion.Rationale != nil {
			question.Rationale = *sourceQuestion.Rationale
		}
		for j, sourceOption := range *sourceQuestion.Options {
			if sourceOption.Option == nil {
				return Exam{}, fmt.Errorf(`question %d option %d field "option" is required`, i+1, j+1)
			}
			if sourceOption.Correct == nil {
				return Exam{}, fmt.Errorf(`question %d option %d field "correct" is required`, i+1, j+1)
			}
			if sourceOption.Rationale == nil {
				return Exam{}, fmt.Errorf(`question %d option %d field "rationale" is required`, i+1, j+1)
			}
			question.Options[j] = Option{
				Option:    *sourceOption.Option,
				Correct:   *sourceOption.Correct,
				Rationale: *sourceOption.Rationale,
			}
		}
		exam.Questions[i] = question
	}
	return exam, nil
}

func validateExam(name string, questions []Question) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("exam name is empty")
	}
	if len(questions) == 0 {
		return errors.New("exam has no questions")
	}

	for i, question := range questions {
		if strings.TrimSpace(question.Question) == "" {
			return fmt.Errorf("question %d is empty", i+1)
		}
		if len(question.Options) == 0 {
			return fmt.Errorf("question %d has no options", i+1)
		}

		seenOptions := map[string]bool{}
		for j, option := range question.Options {
			label := strings.TrimSpace(option.Option)
			if label == "" {
				return fmt.Errorf("question %d option %d is empty", i+1, j+1)
			}
			if seenOptions[label] {
				return fmt.Errorf("question %d contains duplicate option %q", i+1, label)
			}
			seenOptions[label] = true
			if strings.TrimSpace(option.Rationale) == "" {
				return fmt.Errorf("question %d option %d has an empty rationale", i+1, j+1)
			}
		}
		if countCorrect(question) == 0 {
			return fmt.Errorf("question %d has no correct options", i+1)
		}
	}
	return nil
}

func cloneQuestions(source []Question) []Question {
	cloned := make([]Question, len(source))
	for i, question := range source {
		cloned[i] = question
		cloned[i].Options = append([]Option(nil), question.Options...)
	}
	return cloned
}

func countCorrect(question Question) int {
	count := 0
	for _, option := range question.Options {
		if option.Correct {
			count++
		}
	}
	return count
}

func shuffleQuestions(questions []Question) error {
	for i := len(questions) - 1; i > 0; i-- {
		j, err := randomInt(i + 1)
		if err != nil {
			return err
		}
		questions[i], questions[j] = questions[j], questions[i]
	}
	return nil
}

func shuffleOptions(options []Option) error {
	for i := len(options) - 1; i > 0; i-- {
		j, err := randomInt(i + 1)
		if err != nil {
			return err
		}
		options[i], options[j] = options[j], options[i]
	}
	return nil
}

func randomInt(upperBound int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(upperBound)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
