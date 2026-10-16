# Ballerina UDP Library

## Overview

The Ballerina UDP library provides low-level, connectionless (and connection-oriented) UDP client and server (listener) APIs, letting programs exchange datagrams over a socket without any built-in framing or protocol. A `udp:Listener` binds to a local port and dispatches every received datagram to an attached service's `onBytes`/`onDatagram` methods, whose return value (or an explicit `udp:Caller` call) sends a reply. The Go Native Interpreter currently supports the connectionless client, the connection-oriented client, and the listener/service/caller surface for a listener bound to a single local address.

## Key Functionalities

- Send and receive datagrams from/to arbitrary remote hosts with a connectionless `udp:Client`, optionally binding to a local interface, with a configurable read timeout.
- Exchange bytes with a single fixed remote peer using a connection-oriented `udp:ConnectClient`, with outbound payloads larger than a single datagram automatically fragmented across multiple datagrams.
- Start a UDP listener, attach a single service to it, and dispatch every received datagram to the service's `onBytes`/`onDatagram` methods, automatically replying with whatever bytes or `Datagram` the handler returns.
- Reply explicitly from inside a handler using the `udp:Caller` passed to it, either to the datagram's sender or to an arbitrary destination.
- Route listener-level errors (e.g. a socket read failure) to the service's `onError` method.

## Examples

```ballerina
import ballerina/udp;
import ballerina/io;

service class EchoService {
    *udp:Service;

    remote function onBytes(readonly & byte[] data, udp:Caller caller) returns udp:Error? {
        check caller->sendBytes(data);
    }
}

public function main() returns error? {
    udp:Listener echoListener = check new (9000);
    check echoListener.attach(new EchoService());
    check echoListener.'start();

    udp:ConnectClient socketClient = check new ("localhost", 9000, {});
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
| Connectionless client initialization with local interface binding and a read timeout | Supported | |
| Sending and receiving datagrams on the connectionless client | Supported | |
| Closing the connectionless client | Supported | |
| Connection-oriented client initialization with local interface binding and a read timeout | Supported | |
| Sending and receiving bytes on the connection-oriented client | Supported | Payloads larger than a single datagram are fragmented across multiple datagrams on write. |
| Closing the connection-oriented client | Supported | |
| Listener initialization bound to a local interface | Supported | |
| Listener bound to a single fixed remote peer | Not Yet Supported | jBallerina's `ListenerConfiguration.remoteHost`/`remotePort` (a listener that only receives from one remote peer) is not yet implemented in this port. |
| Attaching and detaching a service on a listener | Supported | Only one service may be attached to a listener at a time. |
| Starting a listener | Supported | |
| Stopping a listener gracefully | Supported | |
| Stopping a listener immediately | Supported | |
| Dispatching a received datagram to a service | Supported | Parameters are bound by type, so the bare and `Caller` forms work in either order, as in jBallerina. |
| Validating service methods | Partially Supported | The rules jBallerina's compiler plugin enforces (UDP_101–UDP_105) are checked when the service is attached instead of at compile time; see Notable Behavioural Changes. Any object-typed parameter is accepted as the `Caller` without checking that it is `udp:Caller`. |
| Automatically replying with bytes or a datagram returned from a handler | Supported | |
| Dispatching listener-level errors to the service | Supported | |
| Caller send operations | Supported | |
| Declaring a listener service inline | Supported | An anonymous `service on new udp:Listener(...) { ... }` body attaches directly. |
| Service as a distinct type | Not Yet Supported | jBallerina declares `Service` as a `distinct service object` type. This port declares it as a plain (non-distinct) `service object` type instead — this interpreter cannot yet attach an anonymous `service on new udp:Listener(...) { ... }` body to a `distinct service object` target type, and supporting the inline declaration style was prioritized over `distinct` typing. |
| Error type | Partially Supported | `distinct` error types are not yet supported, so `Error` is currently an alias for `error`. |

### Notable Behavioural Changes

- **`Client.init`/`ConnectClient.init`/`Listener.init` take a plain default-valued configuration record.** jBallerina declares these as `*ClientConfiguration`/`*ConnectClientConfiguration`/`*ListenerConfiguration` included-record parameters, which allow named-argument-style construction at the call site (e.g. `check new(8080, localHost = "x")`). This interpreter cannot currently resolve an included-record parameter that follows other positional parameters when the calling module also imports a second package, so this port uses a plain default-valued parameter instead (e.g. `ListenerConfiguration config = {}`); call sites pass a record literal instead (`check new(8080, {localHost: "x"})`).
- **`Listener.detach()` validates the given service.** jBallerina's `detach()` clears whatever service is currently attached regardless of the argument passed to it. This port returns an `Error` unless the given service is the one currently attached.
- **`Listener.immediateStop()` actually stops the listener.** jBallerina's `immediateStop()` is an unimplemented no-op stub (per its own documentation), while `gracefulStop()` performs the real socket close. This port makes `immediateStop()` force-close the listener's socket immediately, same as `gracefulStop()` — UDP has no per-connection state to drain, so both reduce to the same operation.
- **Service methods are validated when the service is attached, not at compile time.** jBallerina's compiler plugin rejects an invalid udp service (a non-`remote` handler, neither or both of `onBytes`/`onDatagram`, an unknown remote method, a wrong parameter type or count, or a wrong return type) as a compile error. This port has no compiler plugin, so `Listener.attach()` applies the same rules and returns an `Error` carrying the plugin's message; a module-level `service on new udp:Listener(...)` declaration with an invalid service fails at program start-up.
- **`remoteHost` is the sender's IP address.** jBallerina fills `Caller.remoteHost` and the received `Datagram.remoteHost` from `InetSocketAddress.getHostName()`, which reverse-resolves the address, so a datagram from `127.0.0.1` reports `localhost`. This port reports the address as seen on the socket (e.g. `127.0.0.1`) without a reverse-DNS lookup.
- **`Client->receiveDatagram()` reports the sender's address.** jBallerina fills the returned `Datagram`'s `remoteHost`/`remotePort` from the packet's recipient, which is the client's own local address — a bug in the reference implementation that contradicts the field documentation. This port reports the address of the peer that sent the datagram.
- **`Caller.remoteHost`/`Caller.remotePort` are `final string`/`final int`.** jBallerina declares them as mutable `string?`/`int?` fields defaulting to `()`, but always sets them before a handler runs. This port declares them as never-nil `final` fields; code that reads them as `string?`/`int?` still compiles, but assigning to them does not.
