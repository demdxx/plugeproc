# plugeproc

[![Build Status](https://github.com/demdxx/plugeproc/workflows/run%20tests/badge.svg)](https://github.com/demdxx/plugeproc/actions?workflow=run%20tests)
[![Coverage Status](https://coveralls.io/repos/github/demdxx/plugeproc/badge.svg?branch=master)](https://coveralls.io/github/demdxx/plugeproc?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/demdxx/plugeproc)](https://goreportcard.com/report/github.com/demdxx/plugeproc)
[![GoDoc](https://godoc.org/github.com/demdxx/plugeproc?status.svg)](https://godoc.org/github.com/demdxx/plugeproc)

Nowadays the data processing application it's a complex of simple microprograms and extensions.
In many cases necessary to use some external applications to make processing of data more simpler,
for example video or image processing in many case could be solved by special complex
free or proprietar applications more efficient then with some libraries and components
of the language runtime.

To make this process much easear and safety was writen this extension.

## Structure

By design, all extensions must be stored in the special directory and correctly
defined manifest.

* /microprograms directory/
  * /proc-name
    * /.eproc.json - manifest
    * /proc.(sh,py,exe,bash,etc) - optional, depends on manifest
  * /proc-name.eproc.json - alternative defenition without subdirectory

## Manifest

Manifest provides basic information how to connect and use extension.

```json
{
  "type": "exec | shell",
  "interface": "default | stream",
  "command": "cat | sed '{{regexp}}'",
  "args": [],
  "params": [
    {"name": "regexp"},
    {"name": "data", "type": "binary", "is_input": true}
  ],
  "output":  {"type": "binary"}
}
```

* type - integration type of extension
* interface - interraction between application and extension
* command - execution shell command
* args - arguments of the shell command
* params - list of parameters for command execution
  * name - name of the parameter
  * type - **binary** - for input stream only; **json** - auto converting of input parameter into JSON string
  * is_input - for input stream parameter
* output - type of the output variable
  * type - **binary** requires 4 bites LittleEndian order with size of the response; **line** - one line response with '\n' in the end

```go
  var sResp string
  procs.Get("proc-name").Exec(&sResp, "s/Bash/Perl/", "Bash Scripting Language")
  fmt.Println("Response: " + sResp)
```

> Response: Perl Scripting Language

## TODO

 * Add metafile YAML format support
 * Add support goplugin extensions
 * Add support wasm extensions
 * Add support static libraries extensions
