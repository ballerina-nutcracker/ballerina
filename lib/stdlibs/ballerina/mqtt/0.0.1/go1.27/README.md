# Ballerina MQTT Library

## Overview

This module provides an MQTT v5 client and listener for exchanging messages through an MQTT broker. The full jBallerina `mqtt` module covers a `Client` that publishes messages, subscribes to topics and reads responses as a stream; a `Listener` that subscribes to topics and dispatches incoming messages to an `mqtt:Service` with `onMessage`, `onError` and `onComplete` remote methods; a `Caller` for manual acknowledgements and request/response replies; and connection options including authentication, automatic reconnect, failover server URIs, last-will messages, and TLS/mutual TLS configured from PEM files or `crypto:TrustStore`/`crypto:KeyStore`. Service declarations are also checked at compile time by a compiler plugin.

The Go Native Interpreter supports the client and listener APIs, service dispatch, request/response, and TLS/mutual TLS configured from unencrypted PEM files. Plain connections use `tcp://` broker URIs and TLS connections need `ssl://` URIs; a `secureSocket` with a `tcp://` URI returns an `mqtt:Error`.

## Key Functionalities

- Connect to an MQTT v5 broker with optional username/password, keep-alive, connection timeout, clean start, failover server URIs, automatic reconnect and a last-will message.
- Publish messages with QoS 0, 1 or 2, the retained flag, and response-topic/correlation-data properties, receiving a delivery token once delivery completes.
- Subscribe to topics and topic filters, and read incoming messages from a `stream<mqtt:Message, error?>`.
- Check, drop and re-establish the connection with `isConnected`, `disconnect`, `reconnect` and `close`.
- Receive messages through a `mqtt:Listener` and an `mqtt:Service`, attached dynamically or declared at module level.
- Acknowledge messages manually and reply to request messages through `mqtt:Caller`.
- Secure connections with TLS and mutual TLS using PEM certificate and key files.

## Examples

```ballerina
import ballerina/io;
import ballerina/mqtt;

service on new mqtt:Listener(mqtt:DEFAULT_URL, "responder", "demo/request") {
    remote function onMessage(mqtt:Message message, mqtt:Caller caller) returns error? {
        check caller->respond({payload: ("echo: " + check string:fromBytes(message.payload)).toBytes()});
    }
}

public function main() returns error? {
    mqtt:Client mqttClient = check new (mqtt:DEFAULT_URL, "requester");
    check mqttClient->subscribe("demo/response");
    stream<mqtt:Message, error?> responses = check mqttClient->receive();

    mqtt:DeliveryToken token = check mqttClient->publish("demo/request", {
        payload: "hello".toBytes(),
        properties: {responseTopic: "demo/response", correlationData: "req-1".toBytes()}
    });
    io:println("published message ", token.messageId);

    record {|mqtt:Message value;|}? response = check responses.next();
    if response is record {|mqtt:Message value;|} {
        io:println(check string:fromBytes(response.value.payload));
    }
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

### Client

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Client initialization | Supported | |
| Publishing messages | Supported | |
| Subscribing to topics and topic filters | Supported | Accepts `string`, `string[]`, `mqtt:Subscription` and `mqtt:Subscription[]`. |
| Receiving messages as a stream | Supported | `receive()` returns a `stream<mqtt:Message, error?>`. The interpreter does not yet support `foreach` over a stream, so read it with `next()`. |
| Connection state check | Supported | |
| Disconnecting, reconnecting and closing | Supported | |
| Last-will message | Supported | |

### Listener and Service

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Listener initialization | Supported | |
| Attaching and detaching services | Supported | |
| Starting and stopping the listener | Supported | `'start`, `gracefulStop` and `immediateStop`. |
| Module-level listener and service declarations | Supported | |
| Message dispatch to the service | Supported | `onMessage(mqtt:Message)` and `onMessage(mqtt:Message, mqtt:Caller)`. |
| Error dispatch to the service | Supported | `onError(mqtt:Error)`. |
| Delivery-complete dispatch to the service | Supported | `onComplete(mqtt:DeliveryToken)`. |
| Manual message acknowledgement | Supported | `manualAcks` with `mqtt:Caller->complete()`. |
| Request/response replies | Supported | `mqtt:Caller->respond()`. |
| Compile-time service validation | Partially Supported | The interpreter has no compiler-plugin support, so service method signatures are checked when the service is attached instead: an unknown remote method, a non-`remote` `onMessage`/`onError`/`onComplete`, or a wrong parameter count, parameter type or return type makes `attach` return an `mqtt:Error` with the jBallerina diagnostic message (`MQTT_102`, `MQTT_104`–`MQTT_109`, `MQTT_111`–`MQTT_113`, `MQTT_115`–`MQTT_117`). Parameter types are checked structurally, so a supertype such as `anydata` for the message is accepted. A service without `onMessage` (`MQTT_101`) is still reported at runtime through `onError`, and resource methods (`MQTT_103`) and multiple listener attachments (`MQTT_110`) are not reported. |

### Connection and Security

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Username and password authentication | Supported | |
| Connection options | Supported | `keepAliveInterval`, `connectionTimeout`, `cleanStart`, `serverUris`, `automaticReconnect` and `maxReconnectDelay`. A negative `keepAliveInterval` or `connectionTimeout`, or a `keepAliveInterval` above 65535, returns an `mqtt:Error`. |
| TLS with a trusted certificate file | Supported | `SecureSocket.cert` as a PEM certificate file path. |
| Mutual TLS with a certificate and key file | Partially Supported | `SecureSocket.key` as an `mqtt:CertKey` with an unencrypted PEM key. Encrypted private keys (`keyPassword` on an encrypted key) are not supported yet; see [#1338](https://github.com/ballerina-nutcracker/ballerina/issues/1338). |
| TLS with a trust store or key store | Not Yet Supported | `crypto:TrustStore` as `SecureSocket.cert` and `crypto:KeyStore` as `SecureSocket.key` return an `mqtt:Error`; see [#1338](https://github.com/ballerina-nutcracker/ballerina/issues/1338). |
| TLS protocol selection | Supported | `SecureSocket.protocol`. |
| WebSocket transport | Not Yet Supported | `ws://` and `wss://` broker URIs return an `mqtt:Error`. |

### Errors

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Module error type | Supported | `mqtt:Error` is a `distinct` error carrying the Paho `reasonCode` detail. The interpreter does not yet support `error.detail()`, so the reason code cannot be read from Ballerina code. |

### Notable Behavioural Changes

- **Invalid QoS values and topics return an error instead of panicking.** jBallerina lets Paho's `IllegalArgumentException` escape `publish`, `subscribe`, `Caller->respond`, `Listener.'start` and client initialization (will message), so an out-of-range QoS or a malformed topic name or filter panics with `java.lang.IllegalArgumentException`; the Go-native version returns an `mqtt:Error` with Paho's validation message (or `Invalid QoS value: <n>. Allowed values are 0, 1 and 2` where Paho gives none) — reported as https://github.com/ballerina-platform/ballerina-library/issues/9328.
- **QoS 0 publishes do not consume a message ID.** jBallerina's Paho client assigns a message ID to every publish, so a QoS 0 delivery token carries a non-zero `messageId` and later publishes and subscriptions get the next IDs; the Go-native version follows the MQTT protocol, which assigns no packet identifier to QoS 0 messages, so a QoS 0 delivery token has `messageId` 0 and subsequent IDs are one lower than on jBallerina.
- **Errors printed by the listener have no stack frames.** jBallerina prints an error returned by `onMessage`, `onError` or `onComplete` (or a connection error with no `onError`) as `error: <message> <detail>` followed by the Ballerina call stack; the Go-native version prints only the `error: <message> <detail>` line because error values do not carry a call stack.
- **A panic in onMessage is reported to onError.** jBallerina prints a panic raised by `onMessage` and does not invoke `onError`; the Go-native version passes it to `onError` as an `mqtt:Error` carrying the panic's message (or prints it when the service has no `onError`), as the maintainers specified in https://github.com/ballerina-platform/ballerina-library/issues/9326. An error returned from `onMessage` is still only printed.
- **Caller->respond and Caller->complete report failures.** jBallerina publishes the reply and sends the acknowledgement on a background thread, so both return `()` unless the response topic is missing; the Go-native version completes them before returning and returns an `mqtt:Error` when they fail (for example `Client is not connected`), as the maintainers specified in https://github.com/ballerina-platform/ballerina-library/issues/9326.
- **A rejected subscription returns an error.** jBallerina's Paho client does not check the reason codes in a SUBACK, so `subscribe` and `Listener.'start` return `()` when the broker refuses a topic filter and no messages arrive for it; the Go-native version returns an `mqtt:Error` carrying the first failure reason code (for example `Not authorized.`).
- **Manual acknowledgements are sent in arrival order.** jBallerina acknowledges each message as soon as `Caller->complete()` is called; the Go-native version follows the MQTT 5 rule that QoS 1 and 2 acknowledgements are sent in the order the messages arrived, so with `manualAcks` a message that is never completed holds back the acknowledgements of all later messages for the rest of the connection, even though their `complete()` returns `()`.
- **detach keeps the broker connection.** jBallerina's `Listener.detach` disconnects the listener from the broker; the Go-native version unsubscribes the listener's topics and keeps the connection, so attaching a service again re-subscribes them, as the maintainers specified in https://github.com/ballerina-platform/ballerina-library/issues/9326.
- **A repeated 'start does not subscribe again.** jBallerina's `Listener.'start` subscribes to the listener's topics on every call, so the broker redelivers a retained message to `onMessage` once per `'start`; the Go-native version subscribes only when the topics are not already subscribed, so a second `'start` is a no-op and a retained message is delivered once.
- **TLS certificates are verified more strictly.** jBallerina accepts a server certificate that matches the host only through its Common Name, and a self-signed trusted certificate that has expired; the Go-native version uses Go's TLS verification, which requires the host in the certificate's Subject Alternative Name and rejects expired certificates, failing the connection with `Untranslated MqttException - RC: 0`.
- **Service methods are validated when the service is attached, not at compile time.** jBallerina's compiler plugin rejects an invalid mqtt `service` declaration (an unknown remote method, a non-`remote` handler, or a wrong parameter count, parameter type or return type) as a compile error, but its `attach()` accepts any `service class` value and binds whatever values it dispatches to the parameters. This port has no compiler plugin, so `Listener.attach()` applies the same rules to every service, including `service class` values jBallerina would accept, and returns an `mqtt:Error` carrying the plugin's message; a module-level `service on new mqtt:Listener(...)` declaration with an invalid service fails at program start-up.
- **Certificate and key paths in errors are not made absolute.** jBallerina reports a missing certificate or key file with its absolute path; the Go-native version reports the path as it was configured.
