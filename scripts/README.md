# Scripts

This directory contains small, repeatable development helpers.

## Validation scripts

### Full local validation

```cmd
scripts\test-all.cmd
```

The default validation runs:

- formatting check with `gofmt`;
- all unit/seed tests with `go test`;
- coverage profile generation;
- `go vet`;
- `go build`.

Mutation testing and long-running fuzz sessions are opt-in because they are slower than the normal feedback loop:

```cmd
set RUN_FUZZ=1
set RUN_MUTATION=1
scripts\test-all.cmd
```

### Fuzz testing

```cmd
scripts\test-fuzz.cmd
```

The default fuzz duration is `30s` per target. Override it with:

```cmd
set FUZZ_TIME=2m
scripts\test-fuzz.cmd
```

The current machine must use an architecture supported by Go's coverage-guided fuzzing. The script reports an explicit error on unsupported architectures.

### Mutation testing

Install [Gremlins](https://github.com/go-gremlins/gremlins) and make the `gremlins` executable available on `PATH`, then run:

```cmd
scripts\test-mutation.cmd
```

Mutation testing is limited to `internal/ledger/domain`. The default thresholds are 80% test efficacy and 80% mutant coverage. Override them when experimenting:

```cmd
set MUTATION_EFFICACY_THRESHOLD=70
set MUTATION_COVERAGE_THRESHOLD=70
scripts\test-mutation.cmd
```

The script uses one worker and a larger timeout coefficient by default because parallel mutation processes can contend for Go build-cache files on Windows. `TIMED_OUT` mutants are reported as warnings: Gremlins treats them as mutations that caused the test process to time out, but excludes them from its efficacy calculation. `LIVED` and `NOT COVERED` results remain visible in the JSON report and affect the configured thresholds.
