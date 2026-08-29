package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Session struct {
	ID              string     `json:"id"`
	ExamName        string     `json:"exam_name"`
	ExamPath        string     `json:"exam_path"`
	InstantFeedback bool       `json:"instant_feedback"`
	Questions       []Question `json:"questions"`
	Answers         [][]int    `json:"answers"`
	Current         int        `json:"current"`
}

type sessionStore struct {
	dir string
}

func newSessionStore() (*sessionStore, error) {
	dir, err := sessionsDir()
	if err != nil {
		return nil, err
	}
	return &sessionStore{dir: dir}, nil
}

func (store *sessionStore) save(session *Session) error {
	if err := validateSession(session); err != nil {
		return fmt.Errorf("cannot save session: %w", err)
	}
	path, err := store.path(session.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(store.dir, 0700); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(store.dir, "."+session.ID+"-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	defer temporary.Close()

	if err := temporary.Chmod(0600); err != nil {
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(session); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace session file: %w", err)
	}
	return nil
}

func (store *sessionStore) load(id string) (*Session, error) {
	path, err := store.path(id)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var session Session
	if err := decoder.Decode(&session); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, fmt.Errorf("decode session: %w", err)
		}
		return nil, errors.New("session file contains trailing data")
	}
	if session.ID != id {
		return nil, errors.New("session ID does not match its filename")
	}
	if err := validateSession(&session); err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}
	return &session, nil
}

func (store *sessionStore) remove(id string) error {
	path, err := store.path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (store *sessionStore) path(id string) (string, error) {
	if !validSessionID(id) {
		return "", errors.New("session ID must be a valid UUID v4")
	}
	if strings.TrimSpace(store.dir) == "" {
		return "", errors.New("session directory is empty")
	}
	return filepath.Join(store.dir, id+".json"), nil
}

func validateSession(session *Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	if !validSessionID(session.ID) {
		return errors.New("session ID is invalid")
	}
	if strings.TrimSpace(session.ExamPath) == "" {
		return errors.New("session exam path is empty")
	}
	if err := validateExam(session.ExamName, session.Questions); err != nil {
		return err
	}
	if len(session.Answers) != len(session.Questions) {
		return errors.New("session answer data does not match questions")
	}
	if session.Current < 0 || session.Current > len(session.Questions) {
		return errors.New("session progress is invalid")
	}

	for i, answer := range session.Answers {
		if i >= session.Current {
			if len(answer) != 0 {
				return fmt.Errorf("question %d is unanswered but contains saved selections", i+1)
			}
			continue
		}
		if err := validateSavedAnswer(session.Questions[i], answer); err != nil {
			return fmt.Errorf("question %d answer is invalid: %w", i+1, err)
		}
	}
	return nil
}

func validateSavedAnswer(question Question, selected []int) error {
	required := countCorrect(question)
	if len(selected) != required {
		return fmt.Errorf("expected %d selection(s), found %d", required, len(selected))
	}
	seen := map[int]bool{}
	for _, index := range selected {
		if index < 0 || index >= len(question.Options) {
			return fmt.Errorf("selection index %d is out of range", index)
		}
		if seen[index] {
			return fmt.Errorf("selection index %d appears more than once", index)
		}
		seen[index] = true
	}
	return nil
}

func validSessionID(id string) bool {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	for i, char := range []byte(id) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !isHex(char) {
			return false
		}
	}
	if id[14] != '4' {
		return false
	}
	variant := id[19]
	return variant == '8' || variant == '9' || variant == 'a' || variant == 'b' || variant == 'A' || variant == 'B'
}

func isHex(char byte) bool {
	return char >= '0' && char <= '9' || char >= 'a' && char <= 'f' || char >= 'A' && char <= 'F'
}

func newSessionID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}

	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80

	encoded := hex.EncodeToString(value)
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32]), nil
}

func sessionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".examsim", "sessions"), nil
}
