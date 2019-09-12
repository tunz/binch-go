# binch

A lightweight ELF binary path tool.

![render1567170049118](https://user-images.githubusercontent.com/7830853/64022926-2e990000-cb72-11e9-9736-5c349cc0618f.gif)

## Installation

Using Install script
```
$ curl https://raw.githubusercontent.com/tunz/binch-go/master/install.sh | bash
```

or, you can download a binary in [releases](https://github.com/tunz/binch-go/releases) page.

## Usage

```
$ ./binch [binary name]
```

### Shortcuts

#### Main View
```
g: Go to a specific address. (if not exists, jump to nearest address)
d: Remove a current line. (Fill with nop)
q: Quit.
s: Save a modified binary to a file.
enter: Modify a current line.
j/k: Move to next/previous instruction.
ctrl+f/b: Move to next/previous page.
```

#### Patch View
```
tab: Switch focusing opcode/instruction.
enter: Apply the patch.
```
