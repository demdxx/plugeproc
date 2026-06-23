# plugeproc

[![Build Status](https://github.com/demdxx/plugeproc/workflows/run%20tests/badge.svg)](https://github.com/demdxx/plugeproc/actions?workflow=run%20tests)
[![Coverage Status](https://coveralls.io/repos/github/demdxx/plugeproc/badge.svg?branch=main)](https://coveralls.io/github/demdxx/plugeproc?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/demdxx/plugeproc)](https://goreportcard.com/report/github.com/demdxx/plugeproc)
[![GoDoc](https://godoc.org/github.com/demdxx/plugeproc?status.svg)](https://godoc.org/github.com/demdxx/plugeproc)

**plugeproc** is a Go library for running external programs and Docker containers as typed
procedures with a clean, uniform interface.

Data-processing pipelines often rely on specialised external tools — image converters,
ML inference scripts, shell utilities — that are more practical to call as subprocesses
than to embed in-process. plugeproc gives each such tool a small _manifest_ file
(`.eproc.json` / `.eproc.yaml`) that declares its inputs, outputs, and how to run it.
Your Go code then calls them through a single `Store.Exec` call regardless of whether
the tool is a local script or a Docker container.

## Quick start

```go
import (
    plugeproc "github.com/demdxx/plugeproc"
    decodejson "github.com/demdxx/plugeproc/decode/json"
    decodeyaml "github.com/demdxx/plugeproc/decode/yaml"
    "github.com/demdxx/plugeproc/loader/fs"
)

store, err := plugeproc.NewStoreFromLoader(ctx,
    fs.New("./procs",
        fs.WithDecoder(decodejson.Decoder, decodeyaml.Decoder)))

// One-shot call — result decoded into a struct
var result MyStruct
err = store.Exec(ctx, "my-proc", &result, "param1", param2Value)

// Or via a cached Proc handle
p := store.Get("my-proc")
err = p.Exec(ctx, &result, "param1", param2Value)
```

Parameters are passed positionally and matched to the `params` list in the manifest.

---

## Directory layout

Manifests are discovered by `loader/fs` in two equivalent forms:

```
procs/
├── image-resize/          # directory form
│   ├── .eproc.yaml        # manifest
│   └── image-resize.sh    # script (driver: exec auto-discovers this)
│
├── transcribe.eproc.yaml  # flat-file form (no subdirectory needed)
│
└── stream-proc/
    ├── .eproc.json
    └── run.sh             # fallback script name (run, exec, or main)
```

For `driver: exec`, the loader searches for the script in this order:

1. `{proc-name}.{sh,py,bash,…}` — same name as the directory / manifest
2. `run.{sh,py,bash,…}`
3. `exec.{sh,py,bash,…}`
4. `main.{sh,py,bash,…}`

---

## Manifest reference

Manifests are JSON or YAML files named `.eproc.json` / `.eproc.yaml` (inside a directory)
or `<name>.eproc.json` / `<name>.eproc.yaml` (flat file next to other manifests).

```yaml
# Preferred field names
driver: shell | exec | docker # execution backend  (default: exec)
mode: call  | stream # interaction mode   (default: call)

run: <command> # what to run  (see "run: forms" below)
args: [arg1, arg2, …] # extra fixed arguments appended after run
env:
  KEY: value

params:
  - name: param-name
    type: string | binary | json | file
    stdin: true # pipe this param's value to process stdin

output:
  type: binary | line | json | file
  name: out # macro name when type is "file" (default "out")

docker: # required when driver: docker
  image: alpine:latest
  pull_image: true
  retain_container: false # keep container alive across Exec calls
  remove_after_done: true
  container_name: my-container
```

> **Legacy fields** — `type` (→ `driver`), `interface` (→ `mode`), `is_input` (→ `stdin`),
> `is_tmp_file` (→ `output.type: file`) are still accepted and promoted automatically.

---

## `run:` — three forms

### Array form — argv list

```yaml
run: [echo, "-n", "{{msg}}"]
```

Each element is a separate argument. `{{macro}}` values are automatically shell-quoted so
spaces and special characters are safe.

### String form — bash expression

```yaml
run: echo -n "{{msg}}"
```

The entire string is passed verbatim to `bash -c`. Macro substitution is **plain** (no extra
quoting), so your own quotes are respected. Use this when you need shell operators like pipes
or redirects.

### Multiline block form — bash script

```yaml
run: |
  read -r LINE
  printf '%s:%s' "{{prefix}}" "$LINE"
```

Same as the string form — passed to `bash -c` as a single argument. Ideal for multi-step
logic. Shell variables (`$LINE`) and your own quoting (`"{{prefix}}"`) work as written.

> **Note:** in the string and multiline forms, macro values are inserted without shell quoting.
> Wrap `{{name}}` in your own quotes (`"{{name}}"`) when the value may contain spaces.

---

## Parameter types

| `type`   | How the value is consumed                                             |
| -------- | --------------------------------------------------------------------- |
| `string` | Cast to string, substituted into `{{name}}` macro in `run:` / `args:` |
| `binary` | Piped to the process stdin (requires `stdin: true`)                   |
| `json`   | JSON-marshalled, substituted into `{{name}}` macro                    |
| `file`   | Written to a temp file; its path is substituted into `{{name}}` macro |

When `stdin: true`, the param value is streamed to the process's standard input. Multiple
stdin params are concatenated in declaration order.

---

## Output types

| `type`   | What the library reads back                                                                       |
| -------- | ------------------------------------------------------------------------------------------------- |
| `binary` | All bytes written to stdout                                                                       |
| `line`   | A single newline-terminated line from stdout                                                      |
| `json`   | A single JSON object/array from stdout — decoded into the Go target                               |
| `file`   | The process writes output to a temp file whose path is injected as `{{out}}` (or a custom `name`) |

For `stream` mode, `line` and `json` read one frame per `Exec` call; `binary` reads a
4-byte little-endian length prefix followed by that many bytes.

---

## Drivers

### `exec` — auto-discovered script

```yaml
driver: exec
mode: call # or stream
params:
  - { name: data, type: binary, stdin: true }
output:
  type: binary
```

The loader finds the script automatically (see [Directory layout](#directory-layout)).
The script is run via `bash -c`. This is the simplest driver: write a script, place it
alongside the manifest, and it works.

### `shell` — explicit command

```yaml
driver: shell
run: [sed, "s/{{from}}/{{to}}/g"]
params:
  - { name: from, type: string }
  - { name: to, type: string }
  - { name: data, type: binary, stdin: true }
output:
  type: binary
```

The `run:` field is required. No script file is needed. Commands that are not in `$PATH`
can be specified as absolute paths.

### `docker` — containerised command

```yaml
driver: docker
run: [ffmpeg, -i, /dev/stdin, -vf, "scale={{w}}:{{h}}", -f, image2pipe, -]
params:
  - { name: w, type: string }
  - { name: h, type: string }
  - { name: video, type: binary, stdin: true }
output:
  type: binary
docker:
  image: jrottenberg/ffmpeg:4.4-alpine
  pull_image: true
  remove_after_done: true
```

Each `Exec` call spins up a fresh container (unless `retain_container: true`).

#### Docker stream mode

```yaml
driver: docker
mode: stream
run: [awk, '{ print "ack:" $0; fflush() }']
params:
  - { name: data, type: binary, stdin: true }
output:
  type: line
docker:
  image: alpine:latest
  retain_container: true
```

A single container is created on the first `Exec` call and reused for all subsequent calls.
The container process must read from stdin and write one response line per request.

---

## Interaction modes

### `call` (default) — one-shot

A new process (or container) is started for every `Exec` call. Suitable for stateless tools.

### `stream` — persistent process

The process is started once and kept alive. Each `Exec` call writes to stdin and reads one
framed response. Suitable for high-throughput scenarios where process startup cost matters.

---

## Go API

```go
// Store — registry of procedures loaded from a directory
store, err := plugeproc.NewStoreFromLoader(ctx, loader)
defer store.Release()

// Execute by name
var out bytes.Buffer
err = store.Exec(ctx, "proc-name", &out, "arg1", "arg2")

// Or hold a typed handle
p := store.Get("proc-name")
if p == nil { /* not found */ }
err = p.Exec(ctx, &out, "arg1", "arg2")

// Decode JSON output directly into a struct
var result MyResult
err = store.Exec(ctx, "json-proc", &result, inputValue)

// Register a manually created proc
p2, err := plugeproc.New(myManifest)
store.Register(p2)
```

### Target types for `Exec`

| Go target type  | Behaviour                          |
| --------------- | ---------------------------------- |
| `io.Writer`     | Output bytes copied directly       |
| `*bytes.Buffer` | Output bytes copied directly       |
| `*string`       | Output decoded as UTF-8 string     |
| any pointer     | Output JSON-decoded into the value |

---

## Complete examples

### Resize an image with ImageMagick (`exec` driver)

```
procs/resize/
├── .eproc.json
└── resize.sh
```

```json
{
  "driver": "exec",
  "args": ["{{width}}", "{{height}}", "{{input}}", "{{out}}"],
  "params": [
    { "name": "width", "type": "string" },
    { "name": "height", "type": "string" },
    { "name": "input", "type": "file" }
  ],
  "output": { "type": "file" }
}
```

```bash
#!/usr/bin/env bash
convert "$3" -resize "${1}x${2}" "$4"
```

```go
var imgBytes []byte
err = store.Exec(ctx, "resize", &imgBytes, "800", "600", originalBytes)
```

### Persistent JSON processor (`exec` + `stream`)

```json
{
  "driver": "exec",
  "mode": "stream",
  "params": [{ "name": "data", "type": "binary", "stdin": true }],
  "output": { "type": "json" }
}
```

```bash
#!/usr/bin/env bash
while IFS= read -r line; do
  echo "$line" | jq '{result: .input, doubled: (.value * 2)}'
done
```

```go
p := store.Get("my-stream-proc")
for _, req := range requests {
    var resp Response
    if err := p.Exec(ctx, &resp, req); err != nil { ... }
}
```

### Inline script with `run:` multiline

```yaml
driver: shell
run: |
  COUNT=$(echo "{{text}}" | wc -w | tr -d ' ')
  printf '{"words":%s}' "$COUNT"
params:
  - { name: text, type: string }
output:
  type: json
```

---

## TODO

- [x] Manifest JSON format (`.eproc.json`)
- [x] Manifest YAML format (`.eproc.yaml`)
- [x] Flat-file manifest form (`name.eproc.json` alongside other manifests)
- [x] `exec` driver — auto-discovered local scripts
- [x] `shell` driver — explicit `run:` command
- [x] `docker` driver — one-shot and stream containers
- [x] `call` and `stream` interaction modes
- [x] Parameter types: `string`, `binary`, `json`, `file`
- [x] Output types: `binary`, `line`, `json`, `file`
- [x] Default script name fallback (`run.*`, `exec.*`, `main.*`)
- [ ] `goplugin` driver — Go plugins
- [ ] `wasm` driver — WebAssembly modules
- [ ] `http` driver — remote HTTP endpoints
