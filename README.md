# Go Sally MVP (Minimum/Minimal Viable Product)
[![Status](https://img.shields.io/badge/status-MVP-orange.svg)]()
[![Go Version](https://img.shields.io/badge/Go-1.24.6-informational)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/akyaiy/GoSally-mvp.svg)](https://pkg.go.dev/github.com/akyaiy/GoSally-mvp)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)

> [!NOTE]
> If you see "💡" in the text, it means the information below is about plans for the future of the project.

## Core features
- **Decentralized nodes**<details>this means that *multiple GS[^1] nodes can be located on a single machine*, provided no attempt is made to disrupt, sabotage, or bypass the built-in protection mechanism against running a node under the same identifier as one already running in the system. Identification plays a role in node communication. 💡 In the future, we plan to create tools for conveniently building distributed systems using node identification.</details>
- **RPC request processing**<details>the GS operates *using HTTP/https and the JSONRPC2.0 protocol.* Unlike gRPC, jsonrpc is extremely simple, allows for easy sending of requests from the browser, and does not require any additional code compilation.</details>
- **Lua script-based methods**<details>*The gopher-lua library is used, providing full support for Lua 5.1.* scripts implement libraries for interacting with sessions (receiving parameters and sending responses), hashing, logging, and more. This allows you to quickly write business logic on the fly without touching the lower layers of abstraction, which also eliminates unnecessary compilation and the risk of breaking the codebase.
  Example of the "echo" method:
  ```lua
  local session = require("internal.session")
  -- import the internal library for interacting with sessions

  session.response.send(session.request.params.get())
  -- send everything passed in the parameters in response.
  ```
  </details>
- **Relatively flexible configuration** <details>
you can configure the server port, address, name, node settings, and more. 💡 More settings are planned in the future.</details>
- ***And more in the future***

> [!IMPORTANT]
> This is the beginning of the project's development, and some aspects of it may be unstable, unfinished, and the text about it may be overly ambitious. It's just a matter of time.

## Quick start
```bash
git clone https://github.com/akyaiy/GoSally-mvp.git && \
cd GoSally-mvp && \
make build && \
echo -e "node:\n  com_dir: \"%path%/com\"" > config.yaml && \
mkdir -p com && \
echo -e 'local session = require("internal.session")\n\nsession.response.send(session.request.params.get())' > com/echo.lua && \
./bin/node run
```

If you have problems, make sure you have all [dependencies](https://github.com/akyaiy/GoSally-mvp/wiki/Getting-started#installing-dependencies) installed, otherwise [file an issue report](https://github.com/akyaiy/GoSally-mvp/issues)

### Test it
```bash
curl -X POST http://localhost:8080/com \
  -d '{"jsonrpc":"2.0","context-version": "v1","method":"echo","params":["Hi!!"],"id":1}'
```
Expected response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": [
    "Hi!!"
  ],
  "data": {
    "responsible-node": "a0e1c440473ffd4d87e32cff2717f5b3",
    "salt": "f26df732-a3be-4400-8e71-b8dc3ba705fc",
    "checksum-md5": "cd8bec6a365d1b8ee90773567cb3ad0a"
  }
}

```
## Concept
The project was originally conceived as a tool for building infrastructure using relatively *small nodes with limited functionality*. 💡 In the future, we plan to create a *web interface for interacting with nodes, administration, and configuration*. The concept is simple: suppose we have a node that manages Bind9. It has all the necessary methods for interacting with the service: creating new zones, viewing zone status, changing configuration, and server operation status. All of this works only through manual configuration, with the exception of larger solutions like Webmin and the BIND DNS Server module. The big problem is that while we only needed web configuration for Bind9, we have to pull in a massive amount of software just to implement one module. What if the service is hosted on a low-power Raspberry Pi? That's where GS nodes come in. By default, GS nodes communicate only through API calls, so 💡 in the future, we plan to create a dedicated, also programmable, web node that will provide convenient access to node management.

There's an obvious advantage here: transparency. The project is *completely open source and aims to support community-driven node functionality*. 💡 In the future, we plan to create a "store" similar to Docker Hub, which will contain scripts for configuring bind9, openvpn, and even custom projects.

## API
As mentioned earlier, *the server handles [jsonrpc2.0](https://www.jsonrpc.org/specification) requests*
```json
{
    "jsonrpc": "2.0",
    "context-version": "v1",
    "method": "test",
    "params": [
        "Hi!!"
    ],
    "id": 1
}
```
This is a typical example of a request using the jsonrpc2.0 protocol.
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        "Hi!!"
    ],
    "data": {
        "responsible-node": "2ad6ebeaf579a7c52801fb6c9dd1b83d",
        "salt": "e7a81115-01c1-45b1-9618-0eae0ff26451",
        "checksum-md5": "cd8bec6a365d1b8ee90773567cb3ad0a"
    }
}
```
In the result field, we see the echo method's response. Those familiar with the jsonrpc2.0 specification will notice that the data structure here is unclear. This is my extension, which has three functions:
| Field | Type | Description |
|--------|------|-------------|
| `responsible-node` | string | ID of the node that executed the task |
| `salt` | string | Random value for each request — can be used to check that the response is unique |
| `checksum-md5` | string | MD5 hash of the result field — can be used to avoid processing identical results separately |

## License
Distributed under the BSD 3-Clause License. See [`LICENSE`](./LICENSE) for more information.

[^1]: Go Sally
