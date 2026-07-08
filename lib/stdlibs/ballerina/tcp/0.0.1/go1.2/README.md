# Ballerina TCP Library

## Overview

The Ballerina TCP library provides low-level, connection-oriented TCP client and server (listener) APIs, letting programs exchange raw bytes over a socket without any built-in framing or protocol. A `tcp:Listener` accepts inbound connections and dispatches each one to an attached service's `onConnect` method, whose returned connection service then receives `onBytes`/`onError`/`onClose` callbacks for that connection. The Go Native Interpreter currently supports the full client and listener/service surface over plaintext or TLS, with certificate-file-based TLS on both sides; PKCS12 trust-store/key-store-based TLS credentials are not yet supported.

## Key Functionalities

- Connect to a remote host with `hostName`/`port`, optionally binding to a local interface, with independent read and write timeouts.
- Send and receive raw bytes over the connection with no framing — each `readBytes()`/`onBytes` call reflects one underlying socket read, exactly as much data as happened to arrive.
- Secure a client connection with a PEM certificate file, or a listener with a certificate and private key file pair, including TLS version selection and cipher suite restriction.
- Start a TCP listener, attach a single service to it, and dispatch every accepted connection through `onConnect` to a dedicated, per-connection service instance.
- Gracefully drain in-flight connections before stopping a listener, or force-stop it and every active connection immediately.

## Examples

```ballerina
import ballerina/tcp;
import ballerina/io;

service class EchoService {
    *tcp:ConnectionService;

    remote function onBytes(tcp:Caller caller, readonly & byte[] data) returns tcp:Error? {
        check caller->writeBytes(data);
    }
}

service class EchoServer {
    *tcp:Service;

    remote function onConnect(tcp:Caller caller) returns tcp:ConnectionService {
        return new EchoService();
    }
}

public function main() returns error? {
    tcp:Listener echoListener = check new (9000);
    check echoListener.attach(new EchoServer());
    check echoListener.'start();

    tcp:Client socketClient = check new ("localhost", 9000, {});
    check socketClient->writeBytes("hello".toBytes());
    readonly & byte[] response = check socketClient->readBytes();
    io:println('string:fromBytes(response));
    check socketClient->close();
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Client initialization with local interface binding and read/write timeouts | Supported | |
| Client secure connection using a certificate file path | Supported | |
| Listener initialization | Supported | |
| Listener secure connection using a certificate and private key file pair | Supported | |
| Client secure connection using a trust store | Cannot Support | Extracting every certificate from a PKCS12 trust store isn't implemented for this module; supplying a trust store value for `ClientSecureSocket.cert` returns an `Error` at runtime. Use a PEM certificate file path instead. |
| Listener secure connection using a key store | Cannot Support | Supplying a key store value for `ListenerSecureSocket.key` returns an `Error` at runtime. Use a certificate and private key file pair instead. |
| TLS version selection and cipher suite restriction | Supported | |
| Sending and receiving raw bytes on the client | Supported | |
| Closing the client connection | Supported | |
| Attaching and detaching a service on a listener | Supported | Only one service may be attached to a listener at a time. |
| Starting a listener | Supported | |
| Stopping a listener gracefully | Supported | |
| Stopping a listener immediately | Supported | |
| Dispatching a new connection to a service | Supported | |
| Dispatching received bytes to a connection service | Partially Supported | Only the `onBytes(Caller, readonly & byte[])` two-parameter form is supported. jBallerina additionally accepts a bare `onBytes(readonly & byte[])` form via reflection-driven parameter binding; this port always invokes the two-parameter form. |
| Automatically returning bytes from a connection service | Supported | |
| Dispatching connection errors | Supported | |
| Dispatching connection close | Supported | |
| Caller write/close operations and connection identity fields | Supported | |
| Declaring a listener service inline | Supported | An anonymous `service on new tcp:Listener(...) { ... }` body attaches directly. |
| Service and connection service as distinct types | Not Yet Supported | jBallerina declares `Service`/`ConnectionService` as `distinct service object` types. This port declares them as plain (non-distinct) `service object` types instead — this interpreter cannot yet attach an anonymous `service on new tcp:Listener(...) { ... }` body to a `distinct service object` target type, and supporting the inline declaration style was prioritized over `distinct` typing. |
| Error type | Partially Supported | `distinct` error types are not yet supported, so `Error` is currently an alias for `error`. |

### Notable Behavioural Changes

- **`Client.init`/`Listener.init` take a plain default-valued configuration record.** jBallerina declares these as `*ClientConfiguration`/`*ListenerConfiguration` included-record parameters, which allow named-argument-style construction at the call site (e.g. `check new("host", 80, localHost = "x")`). This interpreter cannot currently resolve an included-record parameter that follows other positional parameters when the calling module also imports a second package, so this port uses a plain default-valued parameter instead (`ClientConfiguration config = {}` / `ListenerConfiguration config = {}`); call sites pass a record literal instead (`check new("host", 80, {localHost: "x"})`).
- **A connection is closed if `onConnect` returns an error.** jBallerina leaves the connection open with reads permanently paused in this case — a bug in the reference implementation. This port closes the connection instead.
- **`onClose` is invoked exactly once.** jBallerina can invoke `onClose` twice for a locally-initiated `Caller.close()` (once synchronously from `close()`, once again from the resulting disconnect event) — a bug in the reference implementation. This port guards with per-connection state so `onClose` fires exactly once regardless of who triggers the close.
- **`Caller`'s fields are computed once, at accept time.** jBallerina constructs a fresh `Caller` object (recomputing `remoteHost`/`remotePort`/`localHost`/`localPort`/`id`, including a potential reverse-DNS lookup) on every single `onConnect`/`onBytes` dispatch. This port computes these fields once when the connection is accepted and reuses them for its lifetime.
- **`Listener.immediateStop()` actually stops the listener.** jBallerina's `immediateStop()` is an unimplemented no-op stub (per its own documentation). This port force-closes the listener and every active connection immediately.
- **`Listener.detach()` validates the given service.** jBallerina's `detach()` clears whatever service is currently attached regardless of the argument passed to it. This port returns an `Error` unless the given service is the one currently attached.
