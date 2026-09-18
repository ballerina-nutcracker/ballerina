---
name: trace-frontend
description: Gather compilation traces to help with performance analysis of the front-end
---

- IMPORTANT: tracing is only supported with debug builds
```bash
go build -tags debug -o bal-debug ./cli/cmd
```

## Usage
With the `--trace[=<path>]` flag you can record front-end traces as [Chrome Trace Event](https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU) JSON, for a given Ballerina project/file.

Use `--nested` flag to record child spans within each traced phase (requires `--trace`)

```bash
# writes traces.json in the current directory
./bal-debug run --trace corpus/bal/subset1/01-boolean/equal1-v.bal

# custom path, relative to the process working directory or absolute
./bal-debug run --trace=out/run.json myproject

# one span per traced child operation inside each phase, not just the phases
./bal-debug run --trace --nested myproject
```

