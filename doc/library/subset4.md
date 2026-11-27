# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/email` package, starting with an SMTP client that sends plain-text, HTML and multipart messages with attachments over plaintext, implicit TLS or STARTTLS connections.

## [email](https://github.com/ballerina-platform/module-ballerina-email/blob/master/docs/spec/spec.md)

Types: `email:SmtpClient` (client class), `email:SmtpConfiguration`, `email:Message`, `email:Options`, `email:Attachment`, `email:SecureSocket`, `email:Security`, `email:Protocol`, `email:Service`, `email:Error`.

| Method | Notes |
|---|---|
| `new email:SmtpClient(host, username?, password?, *clientConfig)` | Basic authentication with a username and password, or no authentication when both are omitted; any other combination returns `email:Error`. Configuration problems such as an unreadable `secureSocket.cert` are returned as `email:Error`. No connection is made until a message is sent |
| `SmtpClient->sendMessage(email)` | Sends an `email:Message`; a `contentType` that does not contain `text` returns `email:Error` |
| `SmtpClient->send(to, subject, from, body, *options)` | Sends a message built from the arguments and `email:Options` |

Each send opens a new connection, authenticates with `AUTH LOGIN` or `AUTH PLAIN` when the server offers it, and delivers to every To, Cc and Bcc address. A message with only `body` or only `htmlBody` is a single text part, both bodies form a `multipart/alternative`, and attachments follow that body content in a `multipart/mixed`. `contentType`, including its charset, applies to the text body, or to the HTML body when it is the only body. Non-ASCII addresses are sent with the SMTPUTF8 extension, and fail on servers that do not offer it. Headers in `headers` replace generated headers of the same name; a `To`, `Cc` or `Bcc` header replaces that recipient list, and an empty value clears it. Waits on the server are bounded by the RFC 5321 client timeouts. Attachments can be `mime:Entity` values, including nested multipart entities, or `email:Attachment` files sent with their `contentType`.

`security` selects `SSL` (implicit TLS, the default on port 465), `START_TLS_ALWAYS`, `START_TLS_AUTO` or `START_TLS_NEVER`. `secureSocket` supplies the trusted certificates (a PEM bundle or a DER file), TLS versions, cipher suites and host-name verification for whichever TLS the mode uses.

Not covered in this subset: OAuth2 SMTP authentication, the POP3 and IMAP clients, and the POP3 and IMAP listeners.
