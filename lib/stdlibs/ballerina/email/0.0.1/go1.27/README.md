# Ballerina Email Library

## Overview

The Ballerina Email module sends and receives email: an SMTP client sends messages with text and HTML bodies, custom headers and attachments; POP3 and IMAP clients read messages from a mailbox; and POP3 and IMAP listeners poll a mailbox and dispatch new messages to an `email:Service`. Connections can be plaintext, implicit TLS or upgraded with STARTTLS, and SMTP supports basic and OAuth2 authentication. The Go Native Interpreter currently supports the SMTP client with basic and unauthenticated access over every security mode; the OAuth2 SMTP authentication, the POP3 and IMAP clients and the listeners are not yet supported.

## Key Functionalities

- Send email through `email:SmtpClient` with `sendMessage` or `send`, authenticating with a username and password or without authentication.
- Plain-text, HTML and text-plus-HTML (`multipart/alternative`) bodies, with the message `contentType` (including its charset) applied to the text body, or to the HTML body when it is the only body.
- To, Cc, Bcc, Reply-To and Sender addresses, and custom headers that replace generated headers of the same name.
- Attachments from `mime:Entity` values (including nested multipart entities) and from files through `email:Attachment`.
- Implicit TLS (`SSL`), STARTTLS (`START_TLS_ALWAYS`, `START_TLS_AUTO`) and plaintext (`START_TLS_NEVER`) connections, with `email:SecureSocket` trust certificates, TLS versions, cipher suites and host-name verification.

## Examples

```ballerina
import ballerina/email;
import ballerina/mime;

public function main() returns error? {
    email:SmtpClient smtpClient = check new ("smtp.example.com", "alice@example.com", "password",
        port = 587, security = email:START_TLS_ALWAYS);

    mime:Entity report = new;
    report.setText("quarterly figures");

    check smtpClient->send("bob@example.com", "Report", "alice@example.com", "Please find the report attached.",
        htmlBody = "<p>Please find the report attached.</p>", cc = ["carol@example.com"],
        headers = {"X-Priority": "1"}, attachments = report);
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
| SMTP client with username and password | Partially Supported | `email:SmtpClient` authenticates with `AUTH LOGIN` or `AUTH PLAIN`; JavaMail's `DIGEST-MD5` and `NTLM` mechanisms are not supported |
| Unauthenticated SMTP client | Supported | |
| OAuth2 SMTP authentication | Not Yet Supported | `SmtpConfiguration.auth`, `OAuth2GrantConfig`, `OAuth2ClientCredentialsGrantConfig` and `OAuth2PasswordGrantConfig` are not declared yet; they arrive with `ballerina/oauth2` support |
| Send a message record | Supported | `sendMessage` |
| Send with optional parameters | Supported | `send` with `email:Options` |
| Text, HTML and alternative bodies | Supported | |
| Recipient, reply and sender addresses | Supported | |
| Custom headers | Supported | `MIME-Version`, `Content-Type` and `Content-Transfer-Encoding` are generated from the message content, so values given for them in `headers` are not used |
| Attachments from MIME entities | Supported | |
| Attachments from files | Supported | |
| Implicit TLS connections | Supported | |
| STARTTLS connections | Supported | |
| Plaintext connections | Supported | |
| Secure socket configuration | Supported | `cert` accepts a PEM bundle or a single DER certificate file |
| POP3 client | Not Yet Supported | `email:PopClient` and `email:PopConfiguration` are not declared yet |
| IMAP client | Not Yet Supported | `email:ImapClient` and `email:ImapConfiguration` are not declared yet |
| POP3 listener | Not Yet Supported | `email:PopListener` and `email:PopListenerConfiguration` are not declared yet |
| IMAP listener | Not Yet Supported | `email:ImapListener` and `email:ImapListenerConfiguration` are not declared yet |
| Email services | Not Yet Supported | `email:Service` is declared, but no listener can dispatch to it yet |
| Compile-time service validation | Not Yet Supported | The jBallerina compiler plugin's `onMessage`/`onError`/`onClose` checks are not ported yet |
| Static code analysis rules | Cannot Support | The `bal scan` rules for cleartext, opportunistic and weak TLS settings need the jBallerina scanner tooling, which this interpreter does not have |
| Module-level error type | Supported | `email:Error` |

### Notable Behavioural Changes

- **START_TLS_ALWAYS uses STARTTLS.** jBallerina opens an implicit TLS connection for `START_TLS_ALWAYS`, so it cannot reach a STARTTLS submission port; the Go-native version connects in plaintext, upgrades with `STARTTLS` and fails if the server does not offer it (<https://github.com/ballerina-platform/ballerina-library/issues/9729>).
- **Secure socket keeps the security mode.** jBallerina switches to implicit TLS whenever `secureSocket` is set; the Go-native version uses `secureSocket` only as the trust settings for the TLS that the `security` mode selects (<https://github.com/ballerina-platform/ballerina-library/issues/9730>).
- **Optional TLS protocol.** jBallerina panics with a NullPointerException when `secureSocket.protocol` is absent; the Go-native version uses the platform's default TLS settings (<https://github.com/ballerina-platform/ballerina-library/issues/9731>).
- **SMTP initialization errors are returned.** jBallerina panics when the `secureSocket` certificate cannot be read; the Go-native version returns an `email:Error` from `init` (<https://github.com/ballerina-platform/ballerina-library/issues/9732>).
- **Empty TLS versions use the defaults.** jBallerina enables no TLS version when `protocol.versions` is empty, so every handshake fails; the Go-native version uses the platform's default versions (<https://github.com/ballerina-platform/ballerina-library/issues/9733>).
- **Every certificate in the cert file is trusted.** jBallerina trusts only the first certificate in the `secureSocket.cert` file; the Go-native version trusts every certificate in it (<https://github.com/ballerina-platform/ballerina-library/issues/9734>).
- **File attachment content type.** jBallerina ignores `email:Attachment.contentType` and fails the send for a multipart type; the Go-native version uses `contentType` for the attachment and returns an `email:Error` for a multipart type (<https://github.com/ballerina-platform/ballerina-library/issues/9735>).
- **Multipart entity attachments keep their structure.** jBallerina re-sends a multipart `mime:Entity` as `multipart/mixed` without its parameters and headers; the Go-native version keeps its subtype, parameters and headers (<https://github.com/ballerina-platform/ballerina-library/issues/9736>).
- **Multi-valued entity headers.** jBallerina sends only the first value of each `mime:Entity` header; the Go-native version sends every value (<https://github.com/ballerina-platform/ballerina-library/issues/9737>).
- **Message content type with HTML or attachments.** jBallerina ignores `contentType` when the message also has an HTML body or attachments; the Go-native version applies it to the text body part, or to the HTML body when it is the only body (<https://github.com/ballerina-platform/ballerina-library/issues/9738>).
- **Custom headers replace generated headers.** jBallerina adds a second copy of a header such as `Subject` or `From` given in `headers`; the Go-native version replaces the generated header, and a `To`, `Cc` or `Bcc` header replaces that recipient list, an empty value clearing it (<https://github.com/ballerina-platform/ballerina-library/issues/9739>).
- **Header names are validated.** jBallerina writes each key of `headers` as a header name without checking it, so a key such as `Bad Name` goes out as a malformed header line and a line break in a key starts another header; the Go-native version returns `Invalid header name: <name>` for a key that is not an RFC 5322 field name (<https://github.com/ballerina-platform/ballerina-library/issues/9771>).
- **Attachment header names are validated.** jBallerina writes each header name of a `mime:Entity` attachment without checking it, so a line break in a name adds headers to the part; the Go-native version returns `Invalid header name: <name>` for a name that is not an RFC 5322 field name (<https://github.com/ballerina-platform/ballerina-library/issues/9888>).
- **Invalid multipart entity types are returned.** jBallerina panics with a NullPointerException when a multipart `mime:Entity` attachment has a `Content-Type` it cannot parse, such as `multipart/;;`; the Go-native version returns `Invalid content type for the attachment: <type>` (<https://github.com/ballerina-platform/ballerina-library/issues/9889>).
- **Non-multipart types on multipart entities are returned.** jBallerina panics with a ClassCastException when a `mime:Entity` attachment with body parts has a `Content-Type` that is not multipart, such as `text/plain`; the Go-native version returns `Invalid content type for the attachment: <type>` (<https://github.com/ballerina-platform/ballerina-library/issues/9895>).
- **Self-containing multipart entities are returned.** jBallerina panics with a StackOverflow when a `mime:Entity` attachment contains itself through its body parts; the Go-native version returns `A multipart attachment contains itself` (<https://github.com/ballerina-platform/ballerina-library/issues/9904>).
- **Attachment errors are named.** jBallerina fails the send with `IOException while sending message` for a multipart `mime:Entity` attachment with no body parts, a byte body with a multipart Content-Type, or a Content-Transfer-Encoding it does not know; the Go-native version returns `A multipart attachment needs at least one body part`, `Invalid content type for the attachment: <type>` for a multipart byte body whose Content-Type does not parse or has no valid `boundary` parameter (jBallerina sends one that does not parse as a part with no boundary), or `Unsupported Content-Transfer-Encoding: <encoding>` before connecting, and sends a multipart byte body that has a valid boundary as it is. It supports `7bit`, `8bit`, `binary`, `base64` and `quoted-printable`, so it also rejects `uuencode`, which jBallerina sends.
- **No binary transfer encoding over DATA.** jBallerina sends a part labelled `binary` over the SMTP `DATA` command, which cannot carry that encoding (RFC 3030); the Go-native version relabels line-safe data as `7bit` or `8bit` and base64-encodes the rest (<https://github.com/ballerina-platform/ballerina-library/issues/9890>).
- **Long subject words are encoded.** jBallerina writes a subject with more than 998 characters between spaces as one header line beyond the SMTP line limit; the Go-native version writes such a value as RFC 2047 encoded-words, which fold within the limit (<https://github.com/ballerina-platform/ballerina-library/issues/9891>).
- **A 421 reply ends the send.** jBallerina sends the next command after a `421` reply to `EHLO` or `RCPT TO` and reports the failure on the closed connection, `[EOF]`, `Can't send command to SMTP host` or `Exception reading response`; the Go-native version returns the reply, as `421 <text>` for `EHLO` and `Error while sending the message to SMTP server : 421 <text>` for `RCPT TO` (<https://github.com/ballerina-platform/ballerina-library/issues/9897>).
- **Bounded waits on the SMTP server.** jBallerina sets no connection or read timeouts, so an unresponsive server blocks a send forever; the Go-native version applies the RFC 5321 client timeouts (5 minutes for the greeting and each command, 2 minutes for `DATA`, 3 minutes per data write and 10 minutes for the final reply) and returns an `email:Error` when one expires (<https://github.com/ballerina-platform/ballerina-library/issues/9743>).
- **The message passed to sendMessage is not modified.** jBallerina rewrites a single attachment in the caller's `email:Message` into an array, which panics for a readonly message; the Go-native version leaves the message unchanged (<https://github.com/ballerina-platform/ballerina-library/issues/9744>).
- **No empty alternative with attachments.** jBallerina always sends the body of a message with attachments as a `multipart/alternative`, so a message with only `body` gets an empty `text/html` part that HTML clients show as a blank message; the Go-native version sends a single body part, as it does without attachments, and uses `multipart/alternative` only when both bodies are set (<https://github.com/ballerina-platform/ballerina-library/issues/9751>).
- **send checks the content type.** jBallerina's `send` skips the check that `sendMessage` makes and sends a non-text `contentType`; the Go-native version returns `Content type of the email should be text.` from both (<https://github.com/ballerina-platform/ballerina-library/issues/9752>).
- **Only the media type is checked for text.** jBallerina accepts any `contentType` that contains `text`, such as `application/octet-stream; name=text`, and sends the body under that type; the Go-native version checks the media type and returns `Content type of the email should be text.` (<https://github.com/ballerina-platform/ballerina-library/issues/9896>).
- **Content type of a message without a body.** jBallerina ignores `contentType` when neither `body` nor `htmlBody` is set; the Go-native version applies it (<https://github.com/ballerina-platform/ballerina-library/issues/9753>).
- **Charset for text entity attachments.** jBallerina sends a `text/*` `mime:Entity` without a charset as UTF-8 bytes declared as US-ASCII; the Go-native version adds the charset of its bytes, `us-ascii` or `UTF-8` (<https://github.com/ballerina-platform/ballerina-library/issues/9754>).
- **Binary entity attachments are sent intact.** jBallerina sends an entity with a `binary`, `8bit` or `7bit` transfer encoding unencoded, so every bare CR or LF in binary data becomes CRLF; the Go-native version sends such data as base64 (or quoted-printable for text) with the header updated when it is not line-safe (<https://github.com/ballerina-platform/ballerina-library/issues/9755>).
- **Non-ASCII addresses use SMTPUTF8.** jBallerina sends non-ASCII addresses without the SMTPUTF8 extension, writing the envelope address as ISO-8859-1 bytes; the Go-native version sends `SMTPUTF8` with `MAIL FROM` and returns an `email:Error` when the server does not advertise it (<https://github.com/ballerina-platform/ballerina-library/issues/9756>).
- **Non-ASCII message IDs use SMTPUTF8.** jBallerina writes a non-ASCII message ID header from `headers`, such as `In-Reply-To`, as raw UTF-8 without the SMTPUTF8 extension; the Go-native version sends `Message-ID`, `In-Reply-To`, `References` and `Content-ID` unchanged with `SMTPUTF8` and returns `SMTPUTF8 is required for non-ASCII message IDs but host does not support SMTPUTF8` when the server does not advertise it (<https://github.com/ballerina-platform/ballerina-library/issues/9905>).
- **8-bit content follows 8BITMIME.** jBallerina sends a part with an `8bit` or `binary` transfer encoding as raw 8-bit data without `BODY=8BITMIME`, even to a server that does not support 8BITMIME; the Go-native version sends `BODY=8BITMIME` when the server advertises it and otherwise re-encodes those parts as quoted-printable or base64 (<https://github.com/ballerina-platform/ballerina-library/issues/9807>).
- **TLS versions form a range.** jBallerina enables exactly the listed `protocol.versions`; the Go-native version enables every version from the lowest to the highest listed — Go's TLS library accepts only a version range.
- **TLS settings are checked at initialization.** jBallerina accepts any `protocol.versions` or `ciphers` entry and fails the handshake when sending; the Go-native version returns `Unsupported TLS protocol version: <name>` from `init`, skips listed cipher suites that Java supports and Go's TLS library lacks, returns `Unsupported cipher suite: <name>` for any other unknown name or when no listed suite is usable, and returns `The cipher suites cannot be used with the TLS protocol versions` when no listed suite runs within the enabled versions — it validates them when the client is created.
- **TLS 1.3 cipher suites cannot be restricted.** jBallerina enables only the listed TLS 1.3 cipher suites; the Go-native version returns an `email:Error` from `init` when `ciphers` lists some but not all three TLS 1.3 suites and TLS 1.3 is enabled — Go's TLS library always enables all of them. A list without a TLS 1.3 suite disables TLS 1.3.
- **Content type and charset error messages.** jBallerina reports an invalid `contentType` with the MIME parser's message, such as `In Content-Type string <text>, expected '/', got null`, and an unknown charset as `IOException while sending message`; the Go-native version reports `Invalid content type: <type>` and `Unsupported charset: <charset>` — Go's MIME parser reports different details.
- **Address error messages.** JavaMail reports several specific address parse messages; the Go-native version reports JavaMail's `Local address contains control or whitespace`, `Local address starts with dot`, `Local address contains dot-dot` and `Local address ends with dot` for an unquoted local part, and `Illegal address` for any other address Go's parser rejects — Go's address parser reports different details.
