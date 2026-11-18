# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/email` package: an SMTP client that sends plain-text, HTML and multipart messages with attachments, POP3 and IMAP clients that receive them, and POP3 and IMAP listeners that dispatch them to services, over plaintext, implicit TLS or STARTTLS connections.

## [email](https://github.com/ballerina-platform/module-ballerina-email/blob/master/docs/spec/spec.md)

Types: `email:SmtpClient`, `email:PopClient` and `email:ImapClient` (client classes), `email:PopListener` and `email:ImapListener` (listener classes), `email:SmtpConfiguration`, `email:PopConfiguration`, `email:ImapConfiguration`, `email:PopListenerConfiguration`, `email:ImapListenerConfiguration`, `email:Message`, `email:Options`, `email:Attachment`, `email:SecureSocket`, `email:Security`, `email:Protocol`, `email:Service`, `email:Error`.

| Method | Notes |
|---|---|
| `new email:SmtpClient(host, username?, password?, *clientConfig)` | Basic authentication with a username and password, or no authentication when both are omitted; any other combination returns `email:Error`. Configuration problems such as an unreadable `secureSocket.cert` are returned as `email:Error`. No connection is made until a message is sent |
| `SmtpClient->sendMessage(email)` | Sends an `email:Message`; a `contentType` that does not contain `text` returns `email:Error` |
| `SmtpClient->send(to, subject, from, body, *options)` | Sends a message built from the arguments and `email:Options` |
| `new email:PopClient(host, username, password, *clientConfig)` | Connects and authenticates with `USER`/`PASS` to check the server and credentials; a failure returns `email:Error` |
| `PopClient->receiveMessage(folder, timeout)` | Returns the first message in the mailbox and deletes it, checking again after 1 second and then at doubling intervals of up to 30 seconds until `timeout` seconds pass, and returning `()` if none arrives; only `INBOX` (in any case) is a valid folder. A message the server refuses to delete is still returned. `timeout` bounds the waits between checks, not an exchange with the server, which has its own 5-minute limit |
| `PopClient->close()` | Closes the client; later reads, and reads waiting between checks, return `email:Error`, while a read already exchanging with the server finishes and returns its message |
| `new email:ImapClient(host, username, password, *clientConfig)` | Connects and authenticates with `AUTHENTICATE PLAIN`, or `LOGIN` when the server does not offer it, to check the server and credentials; a failure returns `email:Error` |
| `ImapClient->receiveMessage(folder, timeout)` | Returns the oldest message in `folder` that is neither `\Seen` nor `\Deleted` and marks it `\Seen`, waiting as `PopClient->receiveMessage` does; a folder that does not exist or opens read-only returns `email:Error` |
| `ImapClient->close()` | Closes the client as `PopClient->close()` does |
| `new email:PopListener(config)`, `new email:ImapListener(config)` | Creates the listener's `PopClient` or `ImapClient`, which connects to check the server; a failure returns `email:Error` |
| `start()` | Polls `INBOX` every `pollingInterval` seconds while an attached service declares `onMessage`; an interval below one millisecond returns `email:Error`. Starting a started listener does nothing, and a restarted one polls only after its last poll ends |
| `attach(service, name)`, `register(service, name)`, `detach(service)` | Attaches or detaches an `email:Service`; the name is ignored |
| `gracefulStop()`, `immediateStop()` | Stops polling and calls `onClose(())` on each attached service; `gracefulStop` first waits for the running poll and its service calls, while `immediateStop` and a stop from a service method return at once and `onClose` follows the current message |

Each send opens a new connection, authenticates with `AUTH LOGIN` or `AUTH PLAIN` when the server offers it, and delivers to every To, Cc and Bcc address. A message with only `body` or only `htmlBody` is a single text part, both bodies form a `multipart/alternative`, and attachments follow that body content in a `multipart/mixed`. `contentType`, including its charset, applies to the text body, or to the HTML body when it is the only body. Non-ASCII addresses are sent with the SMTPUTF8 extension, and fail on servers that do not offer it. Headers in `headers` replace generated headers of the same name; a `To`, `Cc` or `Bcc` header replaces that recipient list, and an empty value clears it. Waits on the server are bounded by the RFC 5321 client timeouts. Attachments can be `mime:Entity` values, including nested multipart entities, or `email:Attachment` files sent with their `contentType`.

Each POP3 or IMAP read is its own session. An IMAP read keeps its session while it waits, fetches the message with `BODY.PEEK[]`, marks it `\Seen` only after converting it, and never closes the folder with `CLOSE`, so messages flagged `\Deleted` are not expunged.

Each listener poll reads messages until the folder is empty, or until a message cannot be deleted or marked seen, and calls `onMessage` of every attached service for each one, in order and one call at a time; a read error goes to `onError`, or to stderr when no service declares it, and ends the poll. Methods a service does not declare are skipped, and a panic in one or an error it returns is written to stderr.

A received message has `body` from its first `text/plain` part and `htmlBody` from its first `text/html` part not marked as attachments, also inside `multipart/alternative`, `multipart/mixed` and `multipart/related`, decoded in their charsets; other parts are `mime:Entity` attachments with their headers, a message whose top-level MIME structure cannot be parsed has its raw body as `body`, and a nested part whose content type cannot be parsed is an `application/octet-stream` attachment. Addresses are returned as JavaMail formats them, `replyTo` falls back to the From addresses, and `headers` keeps the last value of a repeated header.

`security` selects `SSL` (implicit TLS, the default, on port 465 for SMTP, 995 for POP3 and 993 for IMAP), `START_TLS_ALWAYS`, `START_TLS_AUTO` or `START_TLS_NEVER`. `secureSocket` supplies the trusted certificates (a PEM bundle or a DER file), TLS versions, cipher suites and host-name verification for whichever TLS the mode uses.

Not covered in this subset: OAuth2 SMTP authentication, and the compile-time validation of email services.
