# examsim

`examsim` is a small command-line Exam Simulator written in Go. It loads an exam from a YAML file, presents questions in random order, shuffles answer options, supports single-answer and multiple-answer questions, and prints a score summary at the end.

## Features

- Load exams from YAML with strict schema validation.
- Randomize question order for each new session.
- Randomize option order per question.
- Support questions with one or more correct answers.
- Require the exact number of answers for multi-answer questions.
- Show optional instant feedback with rationales for incorrect answers or for every answer.
- Run a random subset of a larger question bank.
- Save interrupted sessions and resume them later.

## Installation

### Quick install on Linux and macOS

```sh
curl -fsSL https://raw.githubusercontent.com/algonc/examsim/main/install.sh | sh
```

The installer verifies the latest stable release archive's SHA-256 checksum and installs `examsim` in `~/.local/bin`. Follow the printed hint if that directory needs to be added to your `PATH`. Rerun the command to upgrade, then run `examsim -help` to verify the installation.

Pass `--version` to select a release and `--bin-dir` to set the destination. Both options are optional.

```sh
curl -fsSL https://raw.githubusercontent.com/algonc/examsim/main/install.sh | sh -s -- --version v1.2.0 --bin-dir "$HOME/.local/bin"
```

The installer copies only the executable. For the sample exam used below, download a release package or save [examples/exam1.yaml](examples/exam1.yaml) separately.

### Release binaries

Download the package for your operating system and architecture from the [latest GitHub release](https://github.com/algonc/examsim/releases/latest). Extract the archive, then place the `examsim` executable in a directory included in your `PATH`.

Release assets are available for Linux, macOS, and Windows on AMD64 and ARM64. Compare the archive's SHA-256 digest with its entry in `checksums.txt` before installing it.

### Go

Use Go 1.26.0 or newer to install from source.

```sh
go install github.com/algonc/examsim/cmd/examsim@latest
```

## Usage

```sh
examsim -e <exam.yaml> [options]
examsim -resume <session-id> [-w <columns>]
examsim -help
```

## Options

| Option | Description |
| --- | --- |
| `-e <path>` | Load an exam YAML file and start a new session. |
| `-q <count>` | Use a random subset of questions from the exam. New sessions only. |
| `-w <columns>`, `--output-width <columns>` | Set and save output width (minimum 8 columns). Use `0` for automatic sizing, with an 80-column fallback. With the flag omitted, new sessions use automatic sizing and resumed sessions reuse their saved setting. |
| `-i`, `--instant-feedback`, `--instant-feedback=incorrect` | Show correctness after each answer and rationales for incorrect answers. New sessions only. |
| `-a`, `--instant-feedback=all` | Show correctness and available rationales after every answer. New sessions only. |
| `-resume <session-id>` | Resume a saved session. |
| `-h`, `-help`, `--help` | Show help and exit. |

## Examples

Instant feedback is off unless you pass a feedback flag.

Start an exam:

```sh
examsim -e examples/exam1.yaml
```

Show correctness after each answer and rationales for incorrect answers:

```sh
examsim -e examples/exam1.yaml --instant-feedback
```

Show rationales after correct and incorrect answers:

```sh
examsim -e examples/exam1.yaml -a
```

Use only a random subset of questions:

```sh
examsim -e examples/exam1.yaml -q 50
```

Resume an interrupted session:

```sh
examsim -resume 0bd73aa1-af51-45cd-af81-544d65239a4b
```

Start with a fixed width, change it when resuming, or reset it to automatic sizing:

```sh
examsim -e examples/exam1.yaml -w 100
examsim -resume 0bd73aa1-af51-45cd-af81-544d65239a4b --output-width 60
examsim -resume 0bd73aa1-af51-45cd-af81-544d65239a4b -w 0
```

Print help:

```sh
examsim -help
```

## Exam Format

Exams are YAML files with a top-level `name` and a `questions` list. Each question requires a `question` prompt and an `options` list, and can also include an optional `rationale` explaining the whole question. Unknown fields are rejected so mistakes in an answer key do not pass silently.

Both question-level and option-level `rationale` fields are optional and may be omitted or left blank. When provided, the question rationale appears before the option rationales. In `all` mode, rationales appear immediately after each answer. In `incorrect` mode, they appear immediately after incorrect answers. The optional detailed report at the end shows rationales for incorrect answers in any mode.

Each option has these fields:

- `option`: the displayed answer text
- `correct`: `true` or `false`
- `rationale` (optional): an explanation of why the option is correct or incorrect

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
┌──────────────────────────────────────────────────────────────────────────────┐
│ Question 2 of 2                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ What planet has rings? (Select 2 answers.)                                   │
│                                                                              │
│ 1 - Uranus                                                                   │
│                                                                              │
│ 2 - Saturn                                                                   │
│                                                                              │
│ 3 - Sun                                                                      │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
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

The saved session includes the shuffled question and option order, rationales, answers so far, current position, instant feedback mode, and any fixed output width. Older sessions without a saved width use automatic sizing. Resume IDs are UUIDs and can only address files inside the sessions directory. Options that configure a new exam, such as `-q`, `-i`, `-a`, and `--instant-feedback`, cannot be combined with `-resume`.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for source builds, tests, and contribution guidelines. Maintainers can find the release procedure in [RELEASING.md](RELEASING.md).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
