# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with byte-stream and Server-Sent Event bodies for the `http` module.

## [http](https://github.com/ballerina-platform/module-ballerina-http/blob/master/docs/spec/spec.md)

Subset 4 adds byte-stream request and response bodies and Server-Sent Events to the client and service surface of [subset 3](subset3.md#http). `TargetType` is now `typedesc<http:Response|anydata|stream<http:SseEvent, error?>>`, `RequestMessage` is now `anydata|http:Request|mime:Entity[]|stream<byte[], io:Error?>`, and `setPayload` accepts `stream<byte[], io:Error?>` on both `http:Request` and `http:Response`, plus `stream<http:SseEvent, error?>` on `http:Response`, dispatching to `setByteStream` or `setSseEventStream`.

### Byte streams

Request and response bodies can be sent and read as `stream<byte[], io:Error?>`:

| Feature | Notes |
|---|---|
| `setByteStream` | Available on both `http:Request` and `http:Response`. Defaults the Content-Type to `application/octet-stream`, keeping an already-set Content-Type when no override is passed. A request body is sent chunked. A response body that ends within 8 KB is sent whole with its `Content-Length`, as in jBallerina; a longer one is sent chunked, block by block as the stream produces it |
| `getByteStream` | Reads a body received from the network, filling every array except the last to `arraySize` (default 8192). Returns an `error` for a non-positive `arraySize`, for a body set by a payload setter or already read by a payload getter, and for an empty or multipart body |
| `setPayload` | Accepts a byte stream on both `http:Request` and `http:Response` |
| `RequestMessage` | Now `anydata\|http:Request\|mime:Entity[]\|stream<byte[], io:Error?>`, so a byte stream can be passed directly to `post`, `put`, `patch`, `delete`, and `execute` |

```ballerina
stream<byte[], io:Error?> upload = check io:fileReadBlocksAsStream("./data.bin");
http:Response res = check c->post("/upload", upload);
stream<byte[], io:Error?> download = check res.getByteStream();
```

### Server-Sent Events

A resource function may return `stream<http:SseEvent, error?>` (or `stream<http:SseEvent, error>`) to serve Server-Sent Events. Each event is flushed as soon as it is produced, the response is sent as `text/event-stream` with `Cache-Control: no-cache,public` unless one is already set, and an error from the event stream ends it with a final `error` event. `http:Response` also has `setSseEventStream`, and `setPayload` accepts an event stream.

On the client, a `stream<http:SseEvent, error?>` target, or `getSseEventStream()` on a received response, reads a `text/event-stream` body as `http:SseEvent` records, handing each event on as soon as it is complete:

```ballerina
stream<http:SseEvent, error?> events = check c->get("/events");
record {|http:SseEvent value;|}? event = check events.next();
while event is record {|http:SseEvent value;|} {
    io:println(event.value.data);
    event = check events.next();
}
```
