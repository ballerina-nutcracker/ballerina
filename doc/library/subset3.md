# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) with the `tcp` and `udp`
modules: low-level, connection-oriented and (for `udp`) connectionless socket
client and server APIs, with no built-in framing or protocol.

## [tcp](https://github.com/ballerina-platform/module-ballerina-tcp/blob/master/docs/spec/spec.md)

Raw byte-stream sockets over plaintext or TLS.

### Client

| Feature | Notes |
|---|---|
| `new tcp:Client(host, port, config?)` | `ClientConfiguration`: `localHost`, `timeout` (default 300s), `writeTimeout` (default 300s), `secureSocket` |
| `writeBytes` / `readBytes` | Exchange raw bytes with no framing; each `readBytes()` call reflects one underlying socket read |
| `close` | Closes the connection |
| `secureSocket` (client TLS) | PEM certificate file path trust (`cert`), TLS version selection and cipher suite restriction (`protocol`, `ciphers`), handshake timeout |

### Listener and service

| Feature | Notes |
|---|---|
| `new tcp:Listener(port, config?)` | `ListenerConfiguration`: `localHost`, `secureSocket` |
| `secureSocket` (listener TLS) | Certificate and private key file pair (`key`: `CertKey`), TLS version selection, cipher suite restriction |
| `attach` / `detach` | One service per listener; `detach` validates the given service is the one currently attached |
| `'start` / `gracefulStop` / `immediateStop` | Lifecycle driven by the module's `$start`/`$gracefulStop`/`$immediateStop` hooks; `immediateStop` force-closes the listener and every active connection immediately |
| `onConnect(Caller)` | Dispatched per accepted connection; returns a `ConnectionService` (or `Error?`) — a connection is closed if this errors |
| `onBytes(Caller, readonly & byte[])` | Dispatched per read on the connection; a returned `byte[]` is written back automatically |
| `onError` / `onClose` | Dispatched on a connection read error / on connection close (fires exactly once regardless of who triggers it) |
| `Caller` | `writeBytes`, `close`, and `remoteHost`/`remotePort`/`localHost`/`localPort`/`id` fields, computed once at accept time |
| Declaring a listener service inline | `service on new tcp:Listener(...) { ... }` attaches directly, alongside the named-service-class style |

`tcp:Error` is a plain `error` alias, and `tcp:Service`/`tcp:ConnectionService`
are plain (non-distinct) `service object` types — `distinct` typing is not yet
supported for either.

## [udp](https://github.com/ballerina-platform/module-ballerina-udp/blob/master/docs/spec/spec.md)

Datagram sockets, connectionless or connection-oriented.

### Client

| Feature | Notes |
|---|---|
| `new udp:Client(config?)` | Connectionless client. `ClientConfiguration`: `localHost`, `timeout` (default 300s) |
| `sendDatagram` / `receiveDatagram` | Send/receive a `Datagram` (`remoteHost`, `remotePort`, `data`) to/from an arbitrary remote host |
| `new udp:ConnectClient(host, port, config?)` | Connection-oriented client, fixed to a single remote peer. `ConnectClientConfiguration`: `localHost`, `timeout` (default 300s) |
| `writeBytes` / `readBytes` | Exchange bytes with the connected peer; payloads larger than a single datagram are fragmented across multiple datagrams on write |
| `close` | Closes the client socket |

### Listener and service

| Feature | Notes |
|---|---|
| `new udp:Listener(port, config?)` | `ListenerConfiguration`: `localHost` |
| `attach` / `detach` | One service per listener; `detach` validates the given service is the one currently attached |
| `'start` / `gracefulStop` / `immediateStop` | Lifecycle driven by the module's `$start`/`$gracefulStop`/`$immediateStop` hooks; UDP has no per-connection state, so both stop variants force-close the listener's socket immediately |
| `onBytes(readonly & byte[], Caller)` / `onDatagram(readonly & Datagram, Caller)` | Dispatched per received datagram; a returned `byte[]`/`Datagram` is sent back automatically |
| `onError` | Dispatched on a listener-level socket read failure |
| `Caller` | `sendBytes`, `sendDatagram`, and `remoteHost`/`remotePort` fields — the sender of the datagram this `Caller` was dispatched for |
| Declaring a listener service inline | `service on new udp:Listener(...) { ... }` attaches directly, alongside the named-service-class style |

`udp:Error` is a plain `error` alias, and `udp:Service` is a plain
(non-distinct) `service object` type — `distinct` typing is not yet supported.
