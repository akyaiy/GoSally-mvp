# Go Sally MVP (Minimum/Minimal Viable Product)

### What is this?
System that allows you to build your own infrastructure based on identical nodes and various scripts written using built-in Lua 5.1, shebang scripts (scripts that start with the `#!` symbols), compiled binaries.

### Features
Go Sally is not viable at the moment, but it already has the ability to run embedded scripts, log slog events to stdout, and handle RPC like requests.

### Example of use
The basic directory tree looks something like this
```
.
├── bin
│   └── node			Node core binary file
├── com
│   ├── echo.lua		
│   ├── _globals.lua	Declaring global variables and functions for all internal scripts (also required for luarc to work correctly)
│   └── _prepare.lua	Script that is executed before each script launch
└── Makefile

3 directories, 4 files
```
Launch by command 
```bash
$ make run
```
or for structured logs
```bash
$ make run
```

Example of GET request to server
```bash
curl -s http://localhost:8080/api/v1/com/echo?msg=Hello
```
Then the response from the server
```json
{
  "answer": "Hello",
  "status": "ok"
}
```

### How to install
**You don't need it now, but you can figure it out with the Makefile**