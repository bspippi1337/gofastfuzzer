# gofastfuzzer

[Short description]
A fast, lightweight fuzzer for Go projects. gofastfuzzer helps discover edge-case inputs and crashes by automatically generating and running inputs against target functions or binaries.

<!-- Optional badges (replace with actual URLs) -->
[![Go Report Card](https://goreportcard.com/badge/github.com/OWNER/REPO)](https://goreportcard.com/report/github.com/OWNER/REPO)
[![Go Reference](https://pkg.go.dev/badge/github.com/OWNER/REPO.svg)](https://pkg.go.dev/github.com/OWNER/REPO)
[![CI](https://github.com/OWNER/REPO/actions/workflows/ci.yml/badge.svg)](https://github.com/OWNER/REPO/actions)

Table of contents
- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
  - [Run as library](#run-as-library)
  - [Run as CLI](#run-as-cli)
  - [Fuzz tests (Go built-in fuzzing)](#fuzz-tests-go-built-in-fuzzing)
- [Configuration](#configuration)
- [Examples](#examples)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Features
- Lightweight and fast input generation
- Easy to integrate with existing tests and targets
- CLI for quick fuzz runs
- Support for Go's native fuzzing harness (where applicable)

## Requirements
- Go 1.18+ (recommended) — for built-in fuzzing support
- Standard build tools (make, git)

## Installation
Choose one of the following options:

1. Install the binary (if released):
   ```
   go install github.com/bspippi1337/gofastfuzzer/cmd/gofastfuzzer@latest
   ```

2. Build from source:
   ```
   git clone https://github.com/bspippi1337/gofastfuzzer.git
   cd REPO
   go build ./cmd/gofastfuzzer
   ```

3. Use as a library:
   ```
   go get github.com/bspippi1337/gofastfuzzer@latest
   ```


## Usage

### Run as CLI
Basic invocation:
```
gofastfuzzer -target ./path/to/target -timeout 30s -corpus ./corpus
```

Common flags:
- `-target` (string): Path to the target binary or package function to fuzz
- `-timeout` (duration): Max time to run a fuzz session (e.g., `30s`, `5m`)
- `-corpus` (string): Directory to read/write corpus inputs
- `-workers` (int): Number of concurrent fuzzing workers
- `-output` (string): Directory to write crashing inputs and logs
- `-seed` (int): Seed for deterministic input generation

Example:
```
gofastfuzzer -target ./cmd/sample -timeout 2m -workers 4 -corpus ./testdata/corpus -output ./fuzz_out
```

### Run as library
Import and call from Go code:
```go
import "github.com/OWNER/REPO/pkg/fuzzer"

func main() {
    cfg := fuzzer.Config{
        Timeout:  2 * time.Minute,
        Workers:  4,
        CorpusDir: "testdata/corpus",
    }
    f := fuzzer.New(cfg)
    f.Run(targetFunc) // targetFunc is the function under test
}
```

### Fuzz tests (Go built-in fuzzing)
If your project uses Go's testing fuzz support, you can run:
```
go test ./... -fuzz=Fuzz -fuzztime=30s
```
Integrate generated corpus inputs into your tests by saving them under `testdata/fuzz` or a specified corpus directory.

## Configuration
Provide a config file (YAML/JSON) or environment variables to configure longer-running runs and tuning parameters. Example `gofuzz.yaml`:
```yaml
timeout: "5m"
workers: 8
corpus_dir: "./corpus"
output_dir: "./crashes"
seed: 42
```

## Examples
- Fuzz a CLI that reads stdin:
  ```
  cat sample_input | gofastfuzzer -target ./bin/cli -timeout 30s
  ```

- Integrate into CI (GitHub Actions):
```yaml
name: Fuzz CI
on: [push, pull_request]
jobs:
  fuzz:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.20
      - name: Build
        run: go build ./cmd/gofastfuzzer
      - name: Run quick fuzz
        run: ./gofastfuzzer -target ./cmd/sample -timeout 30s -workers 2
```

## Development
- Run linters and tests:
  ```
  go vet ./...
  golangci-lint run
  go test ./... -short
  ```

- Run full fuzz suite locally:
  ```
  go test ./... -fuzz=Fuzz -fuzztime=1m
  ```

## Contributing
Contributions are welcome! Please open issues for bugs or feature requests and create PRs for changes. Follow the repository's CONTRIBUTING.md if present.

Checklist for PRs:
- Add tests for new behavior
- Run `gofmt` and linters
- Update README/docs where applicable

## License
Specify your license here, for example:
This project is licensed under the MIT License - see the LICENSE file for details.

## Contact / Maintainers
- Maintainer: bspippi1337 (GitHub)
- Issue tracker: https://github.com/bspippi1337/gofastfuzzer/issues
