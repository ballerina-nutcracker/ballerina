# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/messaging` package, a message store abstraction with an in-memory store and a polling listener that retries failed messages and moves them to a dead letter store.

## [messaging](https://github.com/ballerina-platform/module-ballerina-messaging/blob/main/docs/spec/spec.md)

A store-agnostic API for persisting messages and consuming them through a polling listener, with retries and a dead letter store.

Types: `messaging:Message`, `messaging:Store`, `messaging:StoreService`, `messaging:StoreListenerConfiguration`, `messaging:InMemoryMessageStore` (class), `messaging:StoreListener` (class), `messaging:Error`.

| Function | Notes |
|---|---|
| `InMemoryMessageStore->store(payload)` / `->retrieve()` / `->acknowledge(id, success?)` | Payloads are cloned on store and on retrieve; a retrieved message stays in flight until it is acknowledged, and acknowledging with failure makes it available again |
| `new messaging:StoreListener(store, ...config)` | Validates `maxRetries`, `pollingInterval` and `retryInterval`; returns `messaging:Error` on an invalid value |
| `attach` / `detach` | One `messaging:StoreService` per listener; a module-level service declaration must be written `isolated service on ...`; `detach` with a service that is not attached returns an error and leaves polling running |
| `start` / `gracefulStop` / `immediateStop` | Each poll delivers one message to `onMessage`; a failed message is retried `maxRetries` times, then stored in `deadLetterStore` if one is configured; if no dead letter store is configured or storing fails, the message is acknowledged per `ackWithFailureAfterMaxRetries`; stopping or detaching cancels a message waiting to be retried; `gracefulStop` behaves like `immediateStop` |
