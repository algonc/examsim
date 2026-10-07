# examsim

`examsim` is a small command-line Exam Simulator written in Go. It loads an exam from a YAML file, presents questions in random order, shuffles answer options, supports single-answer and multiple-answer questions, and prints a score summary at the end.

## Features

- Load exams from YAML with strict schema validation.
- Randomize question order for each new session.
- Randomize option order per question.
- Support questions with one or more correct answers.
- Require the exact number of answers for multi-answer questions.
- Show optional instant feedback with wrong-answer rationales.
- Run a random subset of a larger question bank.
- Save interrupted sessions and resume them later.

## Installation

### Release binaries

Download the package for your operating system and architecture from the [latest GitHub release](https://github.com/algonc/examsim/releases/latest). Extract the archive, then place the `examsim` executable in a directory included in your `PATH`.

Release assets are available for Linux, macOS, and Windows on AMD64 and ARM64. Compare the archive's SHA-256 digest with its entry in `checksums.txt` before installing it.

### Go

With Go 1.25 or newer installed:

```sh
go install github.com/algonc/examsim@latest
```

## Usage

```sh
examsim -e <exam.yaml> [options]
examsim -resume <session-id>
examsim -help
```

## Options

| Option | Description |
| --- | --- |
| `-e <path>` | Load an exam YAML file and start a new session. |
| `-q <count>` | Use a random subset of questions from the exam. New sessions only. |
| `-i`, `--instant-feedback` | Show correctness after each question and rationales for incorrect answers. New sessions only. |
| `-resume <session-id>` | Resume a saved session. |
| `-h`, `-help`, `--help` | Show help and exit. |

## Examples

Start an exam:

```sh
examsim -e examples/exam1.yaml
```

Show feedback after each question:

```sh
examsim -e examples/exam1.yaml --instant-feedback
```

The short flag works too:

```sh
examsim -e examples/exam1.yaml -i
```

Use only a random subset of questions:

```sh
examsim -e examples/exam1.yaml -q 50
```

Resume an interrupted session:

```sh
examsim -resume 0bd73aa1-af51-45cd-af81-544d65239a4b
```

Print help:

```sh
examsim -help
```

## Exam Format

Exams are YAML files with a top-level `name` and a `questions` list. Each question requires a `question` prompt and an `options` list, and can also include an optional `rationale` explaining the whole question. Unknown fields are rejected so mistakes in an answer key do not pass silently.

The question rationale appears before the option rationales for incorrect answers, either immediately with instant feedback or in the final summary. It is preserved when saving and resuming a session. Omitted or blank question rationales are not displayed.

Each option requires:

- `option`: the displayed answer text
- `correct`: `true` or `false`
- `rationale`: feedback shown for incorrect answers, either immediately with instant feedback or in the final summary

Example:

```yaml
name: Exam 1
questions:
  - question: What planet bears life?
    options:
      - option: Sun
        correct: false
        rationale: Not even a planet.
      - option: Earth
        correct: true
        rationale: Earth bears life.
      - option: Mars
        correct: false
        rationale: No indication found so far that Mars bears life.
    rationale: >-
      Only Earth bears life, the other options are not correct:
      Sun is not even a planet.
      No life has been found on Mars so far.
  - question: What planet has rings?
    options:
      - option: Sun
        correct: false
        rationale: Not even a planet.
      - option: Saturn
        correct: true
        rationale: Saturn has rings and is famous for them.
      - option: Uranus
        correct: true
        rationale: Despite not being as visible as Saturn's rings, Uranus has a system of rings.
```

When a question has more than one correct option, `examsim` displays the number of required choices:

```text
Question 2 of 2

What planet has rings? (choose 2)

1 - Uranus
2 - Saturn
3 - Sun
```

Answers can be entered with spaces, commas, or semicolons:

```text
1 2
1,2
1; 2
```

Standard YAML quoted strings and block scalars are supported. For example:

```yaml
question: >
  Which planet is known
  to bear life?
```

## Sessions

The initial session is saved before the first question, then atomically updated after every answer. If the program receives `Ctrl+C`, or `SIGTERM`/`SIGHUP` on Unix, it preserves the latest completed answer and prints a resume command:

```text
Exiting. Resume this session with:

examsim -resume 0bd73aa1-af51-45cd-af81-544d65239a4b
```

Session files are stored as JSON under:

```text
~/.examsim/sessions/
```

The saved session includes the shuffled question order, shuffled option order, answers so far, and current position. Resume IDs are UUIDs and can only address files inside the sessions directory. Options that configure a new exam, such as `-q` and `--instant-feedback`, cannot be combined with `-resume`.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for source builds, tests, and contribution guidelines. Maintainers can find the release procedure in [RELEASING.md](RELEASING.md).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
