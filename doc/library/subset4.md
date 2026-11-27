# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/tcp` package, a raw-bytes TCP client and listener.

## [tcp](https://github.com/ballerina-platform/module-ballerina-tcp/blob/master/docs/spec/spec.md)

Connection-oriented TCP client and listener that exchange raw bytes with no framing.

| Feature | Notes |
|---|---|
| `new tcp:Client(remoteHost, remotePort, config)` | Connects to the remote host, optionally binding to `localHost`, with a `timeout` for reads and a `writeTimeout` for writes |
| `Client->writeBytes(data)` / `Client->readBytes()` / `Client->close()` | Send raw bytes, read the bytes from one socket read, and close the connection |
| `ClientSecureSocket` | TLS on the client with a PEM certificate file in `cert`, `verifyHostName`, plus `protocol` versions and `ciphers`. A `crypto:TrustStore` returns `tcp:Error` |
| `new tcp:Listener(localPort, config)` | Binds on `localHost` (all interfaces by default) |
| `ListenerSecureSocket` | TLS on the listener with a certificate and private key file pair in `key`, plus `protocol` versions and `ciphers`. A `crypto:KeyStore` returns `tcp:Error` |
| `Listener.attach` / `detach` / `'start` / `gracefulStop` / `immediateStop` | One service per listener. `'start` after a stop binds again. `gracefulStop` stops accepting while open connections stay up; `immediateStop` also closes every open connection |
| `tcp:Service.onConnect(caller)` | Called for each accepted connection; returns the `tcp:ConnectionService` that handles it |
| `tcp:ConnectionService.onBytes` / `onError` / `onClose` | `onBytes(readonly & byte[])` or `onBytes(Caller, readonly & byte[])`; returned bytes are written back to the peer; `onError(tcp:Error)`; `onClose()`. All three are optional; without `onBytes`, received bytes are dropped |
| `tcp:Caller` | `writeBytes`, `close`, and the `remoteHost`, `remotePort`, `localHost`, `localPort` and `id` fields |

The service rules that jBallerina's compiler plugin enforces on `service on` declarations are checked at runtime, for service classes as well: `attach` returns `tcp:Error` for an invalid `tcp:Service`, and an invalid connection service returned from `onConnect` is logged and its connection closed. `tcp:Error` is not a `distinct` type. The `Caller` passed to a service does not carry its `tcp:Caller` class type, so `caller is tcp:Caller` on a value held as `any` returns `false` ([#510](https://github.com/ballerina-nutcracker/ballerina/issues/510)).

Not covered in this subset: `ClientSecureSocket.sessionTimeout`, `ListenerSecureSocket.sessionTimeout`, and `CertKey.keyPassword` (`Listener.start()` returns an error for an encrypted private key). `protocol.versions` takes Java's names (`TLSv1` to `TLSv1.3`), and a list that skips a version, such as `["TLSv1.1", "TLSv1.3"]`, returns `tcp:Error`. `ciphers` does not restrict TLS 1.3 suites, and a list naming no suite Go implements returns `tcp:Error` (from the client once the connection is made).
