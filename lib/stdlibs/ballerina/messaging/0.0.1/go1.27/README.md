# Ballerina Messaging Library

## Overview

The `ballerina/messaging` library provides a store-agnostic API for persisting and consuming messages: a `messaging:Store` interface for storing, retrieving and acknowledging messages, an in-memory store implementation, and a `messaging:StoreListener` that polls a store and dispatches each message to an attached `messaging:StoreService`, with retries and an optional dead letter store. The Go Native Interpreter supports the full library API; processing logs are only partially supported because debug-level logs are never emitted.

## Key Functionalities

- Implement custom message stores against the `messaging:Store` interface (`store`, `retrieve`, `acknowledge`)
- Store messages in memory with `messaging:InMemoryMessageStore`, with in-flight tracking and redelivery after a failed acknowledgement
- Poll a store at a configurable interval with `messaging:StoreListener` and dispatch messages to a `messaging:StoreService`
- Retry failed messages a configurable number of times at a configurable interval
- Move messages that still fail to a dead letter store, or acknowledge them with success or failure (`ackWithFailureAfterMaxRetries`)
- Attach, detach, start and stop the listener programmatically or through a module-level `listener` declaration

## Examples

```ballerina
import ballerina/io;
import ballerina/messaging;

final messaging:InMemoryMessageStore orders = new;
final messaging:InMemoryMessageStore failedOrders = new;

listener messaging:StoreListener orderListener = new (orders, {
    pollingInterval: 1,
    maxRetries: 2,
    retryInterval: 1,
    deadLetterStore: failedOrders
});

isolated service on orderListener {
    isolated remote function onMessage(anydata payload) returns error? {
        if payload !is map<anydata> {
            return error("unexpected payload");
        }
        io:println("shipping ", payload["item"]);
    }
}

public function main() {
    orders->store({item: "book"});
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
| Module error type | Supported | `messaging:Error` is a `distinct error` |
| Message record | Supported | `messaging:Message` with a `readonly` `id` |
| Message store interface | Supported | `messaging:Store`; custom stores implement `store`, `retrieve` and `acknowledge` |
| In-memory message store | Supported | `messaging:InMemoryMessageStore`; payloads are cloned on store and on retrieve |
| Message store service | Supported | `messaging:StoreService`; a module-level service declaration must be written `isolated service on ...` because the interpreter does not infer isolation for service declarations |
| Listener configuration and validation | Supported | `messaging:StoreListenerConfiguration`; `init` rejects a negative `maxRetries` and a non-positive `pollingInterval` or `retryInterval` |
| Attaching and detaching a service | Supported | `StoreListener.attach`, `StoreListener.detach`; one service per listener, and `attach` accepts only a nil attach point; a rejected `detach` leaves polling running (see Notable Behavioural Changes) |
| Listener lifecycle | Supported | `StoreListener.start`, `gracefulStop`, `immediateStop`; as in jBallerina, `gracefulStop` stops immediately; stopping cancels pending retries (see Notable Behavioural Changes) |
| Polling and dispatch | Supported | One message per poll, delivered to `onMessage` as a clone of the stored payload |
| Retries | Supported | `maxRetries`, `retryInterval`; a panic in `onMessage` is treated as an error on every attempt (see Notable Behavioural Changes) |
| Dead letter store | Supported | `deadLetterStore`; the message is acknowledged with success once it is stored there |
| Acknowledgement after the last retry | Supported | `ackWithFailureAfterMaxRetries` |
| Processing logs | Partially Supported | Errors are logged with the same messages and `msgId`/`retryAttempt` keys as jBallerina, but `ballerina/log` prints `module=""` and the error in `error("...")` form, and the debug-level logs are never emitted because the log level cannot be configured yet |

### Notable Behavioural Changes

- **A panic on a retry is treated as a failed attempt.** jBallerina traps a panic in `onMessage` only on the first attempt, so a panic on a retry escapes the polling job, skips the remaining retries and the dead letter store, and leaves the message in flight and never acknowledged; the Go-native version traps every attempt, so the message is retried, stored in the dead letter store or acknowledged as configured ([ballerina-library#9371](https://github.com/ballerina-platform/ballerina-library/issues/9371)).
- **A rejected `detach` does not stop polling.** jBallerina unschedules the polling job before checking that the given service is the attached one, so `detach` with a service that is not attached returns an error but leaves the attached service without polling; the Go-native version checks the service first and leaves the listener running ([ballerina-library#9704](https://github.com/ballerina-platform/ballerina-library/issues/9704)).
- **Stopping the listener cancels pending retries.** In jBallerina, `immediateStop`, `gracefulStop` and `detach` only unschedule the polling job, so a message waiting between retries keeps being delivered to `onMessage` and is then acknowledged or dead-lettered after the listener has stopped; the Go-native version cancels it after the current retry sleep and acknowledges it with failure, so the store can deliver it again after a restart ([ballerina-library#9705](https://github.com/ballerina-platform/ballerina-library/issues/9705)).
