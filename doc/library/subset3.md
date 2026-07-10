# Supported ballerina library features

Subset 3 extends the released [subset 2](subset2.md) with the `file`, `ldap`,
`mime`, `tcp`, `udp`, and `uuid` modules, plus two additions to `http` that
depend on `mime` types now available in this subset.

## [file](https://github.com/ballerina-platform/module-ballerina-file/blob/master/docs/spec/spec.md)

File, directory, and path manipulation utilities.

| Feature | Notes |
|---|---|
| `create` / `remove` / `rename` / `copy` | Create, remove (non-recursive and recursive), rename/move, and copy (with `REPLACE_EXISTING`, `COPY_ATTRIBUTES`, `NO_FOLLOW_LINKS` options) files and directories |
| `getMetaData` | File size, modification time, permissions, and type |
| `readDir` | Read directory contents |
| `createTemp` / `createTempDir` | Create a temporary file / directory |
| `test` | Test file/directory properties: `EXISTS`, `IS_DIR`, `IS_SYMLINK`, `READABLE`, `WRITABLE` |
| `getCurrentDir` | Get the current working directory |
| `getAbsolutePath` / `isAbsolutePath` / `basename` / `parentPath` / `normalizePath` / `splitPath` / `joinPath` / `relativePath` | Cross-platform path manipulation |
| `Listener` / `Service` | Directory change listener: attaches a service whose `onCreate`/`onModify`/`onDelete` remote methods are dispatched on filesystem changes under the configured `path` (optionally `recursive`) |

`file:Error`'s `distinct` subtypes (`FileNotFoundError`, `PermissionError`, etc.)
are declared as plain type aliases of `Error` instead — they are structurally
identical at runtime, so `error is file:FileNotFoundError`-style checks don't
narrow. `file:Service` is likewise declared as a plain (non-`distinct`)
`service object {}` marker instead of jBallerina's `distinct service object
{}`. Unlike jBallerina, `gracefulStop()` closes the underlying OS watch
immediately (same as `immediateStop()`) rather than leaving it running until
process exit, and `attach()` returns its "at least one resource required"
validation error through its `error?` return type instead of throwing it.

## [http](https://github.com/ballerina-platform/module-ballerina-http/blob/master/docs/spec/spec.md)

Subsets 1 and 2 covered the basic http client and server. Subset 3 adds two
`Request`/`Response` methods that depend on `mime` types now available in
this subset.

| Feature | Notes |
|---|---|
| `getFormParams()` | `Request`-only. Decodes an `application/x-www-form-urlencoded` body into a `map<string>`, ported from jBallerina's logic (`mime:getMediaType` content-type check, then percent-decoding). Returns an `error` if the `Content-Type` header is missing, invalid, or not `application/x-www-form-urlencoded`. |
| `setPayload(payload, contentType?)` | `Request` and `Response`. Dispatches by runtime type to `setTextPayload`/`setBinaryPayload`/`setJsonPayload`. Accepts `json\|byte[]` rather than jBallerina's full `anydata\|mime:Entity[]\|stream<byte[], io:Error?>\|stream<SseEvent, error?>` union — the `mime:Entity[]` (multipart) and `stream` branches aren't accepted since neither `http` nor `mime` support those payload kinds yet. |

## [ldap](https://github.com/ballerina-platform/module-ballerina-ldap/blob/master/docs/spec/spec.md)

Connect, authenticate, and interact with LDAP directory servers.

| Feature | Notes |
|---|---|
| `new ldap:Client(config)` | Simple bind (`hostName`, `port`, `domainName`, `password`), optionally over TLS with a PEM certificate file, host-name verification, and TLS version selection |
| `add` / `delete` / `modify` / `rename` | Create, delete, update (attribute replace), and rename directory entries |
| `compare` | Compare an entry's attribute value against an assertion value |
| `getEntry` / `search` / `searchWithType` | Retrieve a single entry, or search a subtree, converted into a caller-specified record type or the generic `ldap:Entry` type |
| `close` / `isConnected` | Close the connection and check connection status |
| Binary attribute decoding | `objectGUID`/`objectSid` decoded to canonical string forms; other binary/non-ASCII values are base64-encoded (RFC 2849) |

`ldap:Error` is a plain `error` alias; `distinct` error types are not yet
supported.

## [mime](https://github.com/ballerina-platform/module-ballerina-mime/blob/master/docs/spec/spec.md)

MIME (RFC 2045/2046) media type, content disposition, and entity handling.

| Feature | Notes |
|---|---|
| `getMediaType` / `MediaType` | Parse and construct MIME media types (`primaryType`, `subType`, `suffix`, `parameters`) |
| `getContentDispositionObject` / `ContentDisposition` | Parse and construct content disposition headers |
| `Entity` header management | `setHeader`, `getHeader`, `getHeaders`, `getHeaderNames`, `addHeader`, `removeHeader`, `removeAllHeaders`, `hasHeader` |
| `Entity` content metadata | `setContentType`, `getContentType`, `setContentId`, `getContentId`, `setContentLength`, `getContentLength`, `setContentDisposition`, `getContentDisposition` |
| `Entity` body | `setText`/`getText`, `setJson`/`getJson`, `setByteArray`/`getByteArray`, `setBody(string\|json\|byte[])` |
| `base64Encode` / `base64Decode` | Base64 encode/decode of strings and byte arrays (`Blob` variants included) |
| Media type and header name constants | `APPLICATION_JSON`, `TEXT_PLAIN`, `CONTENT_TYPE`, etc. |

XML and multipart entity bodies (`setXml`/`getXml`, `setBodyParts`/`getBodyParts`)
are not yet supported (require XML and stream support). `mime:Error` and its
subtypes are plain `error` aliases; `distinct` is not yet supported.

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

## [uuid](https://github.com/ballerina-platform/module-ballerina-uuid/blob/master/docs/spec/spec.md)

Generate, validate, and convert UUIDs (RFC 4122).

| Feature | Notes |
|---|---|
| `createType1AsString` / `createType1AsRecord` | Type 1 (time-based) UUID |
| `createType3AsString` / `createType3AsRecord` | Type 3 (MD5 namespace) UUID |
| `createType4AsString` / `createType4AsRecord` / `createRandomUuid` | Type 4 (random) UUID |
| `createType5AsString` / `createType5AsRecord` | Type 5 (SHA-1 namespace) UUID |
| `nilAsString` / `nilAsRecord` | The nil UUID |
| `validate` / `getVersion` | Validate a UUID string and detect its version (`V1`, `V3`, `V4`, `V5`) |
| `toBytes` / `toString` / `toRecord` | Convert between string, byte array, and the structured `Uuid` record |
| Namespace UUID constants | `NAME_SPACE_DNS`, `NAME_SPACE_URL`, `NAME_SPACE_OID`, `NAME_SPACE_X500`, `NAME_SPACE_NIL` |

Type 1 UUIDs use a random 6-byte node identifier (RFC 4122 §4.5) rather than
the host's MAC address. `uuid:Error` is a plain `error` alias; `distinct` is
not yet supported.
