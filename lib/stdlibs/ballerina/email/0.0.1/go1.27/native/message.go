// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package native

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	mimenative "github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/mime/0.0.1/go1.27/native"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	missingFromError = "The 'from' field is mandatory when using SMTP client without authentication. " +
		"Please specify the 'from' address in the email message."
	attachmentReadError = "IOException while sending message"
	dateLayout          = "Mon, 2 Jan 2006 15:04:05 -0700 (MST)"
	maxLineLength       = 998
	foldColumn          = 76
)

type outgoingMessage struct {
	envelopeFrom string
	recipients   []string
	data         []byte
	// data7 is data with every part in a 7-bit transfer encoding, for servers without 8BITMIME
	// (RFC 6152); it is nil when data has no 8-bit bytes.
	data7 []byte
	// utf8 marks non-ASCII addresses, which need the SMTPUTF8 extension (RFC 6531).
	utf8 bool
	// utf8MessageID marks a non-ASCII message ID header, which also needs SMTPUTF8.
	utf8MessageID bool
}

type mimeHeader struct {
	name, value string
}

// mimePart is a MIME entity whose body is already transfer-encoded with CRLF line endings.
type mimePart struct {
	headers []mimeHeader
	body    []byte
}

type messageBuilder struct {
	ctx *extern.Context
	rt  *runtime.Runtime
	// sevenBit re-encodes parts that declare an identity transfer encoding but hold 8-bit data.
	sevenBit bool
	// files keeps each file attachment's bytes, so both encodings of a message use one read.
	files map[*values.Map][]byte
	// nesting holds the multipart entities being converted, to reject an entity that contains itself.
	nesting map[*values.Object]bool
	// utf8MessageID is set when an attachment entity has a non-ASCII message ID header.
	utf8MessageID *bool
}

// buildMessage composes the RFC 5322 message for an email:Message: the body content alone, or a
// multipart/mixed whose first part is the body content followed by the attachments.
func buildMessage(ctx *extern.Context, rt *runtime.Runtime, username string, email *values.Map) (*outgoingMessage, *values.Error) {
	b := messageBuilder{ctx: ctx, rt: rt, files: map[*values.Map][]byte{}, nesting: map[*values.Object]bool{}, utf8MessageID: new(bool)}
	custom, err := customHeaders(email)
	if err != nil {
		return nil, err
	}
	fromValue, _ := stringField(email, "from")
	if fromValue == "" {
		fromValue = username
	}
	senderValue, _ := stringField(email, "sender")
	fields := map[string][]string{
		"from": nonEmpty(fromValue), "sender": nonEmpty(senderValue),
		"reply-to": stringOrList(email, "replyTo"), "to": stringOrList(email, "to"),
		"cc": stringOrList(email, "cc"), "bcc": stringOrList(email, "bcc"),
	}
	addresses := map[string][]*mail.Address{}
	for _, name := range []string{"from", "sender", "reply-to", "to", "cc", "bcc"} {
		list, err := addressHeader(custom, name, fields[name])
		if err != nil {
			return nil, err
		}
		addresses[name] = list
	}
	if len(addresses["from"]) == 0 && fromValue != "" {
		if addresses["from"], err = parseAddresses(fields["from"]); err != nil {
			return nil, err
		}
	}
	if len(addresses["from"]) == 0 {
		return nil, emailError(missingFromError)
	}
	from := addresses["from"][0]

	headers := []mimeHeader{{"Date", rt.Platform().Time.Now().Format(dateLayout)}}
	for _, name := range []string{"From", "Sender", "Reply-To", "To", "Cc"} {
		if list := addresses[strings.ToLower(name)]; len(list) > 0 {
			headers = append(headers, mimeHeader{name, formatAddresses(list)})
		}
	}
	subject, _ := stringField(email, "subject")
	headers = append(headers,
		mimeHeader{"Message-ID", messageID(from.Address)},
		mimeHeader{"Subject", encodeHeaderValue(subject)},
		mimeHeader{"MIME-Version", "1.0"},
	)
	content, err := b.content(email)
	if err != nil {
		return nil, err
	}
	msg := &outgoingMessage{envelopeFrom: addrSpec(from), data: messageData(headers, content, custom)}
	if has8Bit(content.body) {
		b.sevenBit = true
		content7, err := b.content(email)
		if err != nil {
			return nil, err
		}
		msg.data7 = messageData(headers, content7, custom)
	}
	for _, name := range []string{"to", "cc", "bcc"} {
		for _, addr := range addresses[name] {
			msg.recipients = append(msg.recipients, addrSpec(addr))
		}
	}
	for _, list := range addresses {
		for _, addr := range list {
			if _, nonASCII := asciiCounts([]byte(addr.Address)); nonASCII > 0 {
				msg.utf8 = true
			}
		}
	}
	msg.utf8MessageID = *b.utf8MessageID || slices.ContainsFunc(custom, func(h mimeHeader) bool { return isUTF8MessageID(h.name, h.value) })
	return msg, nil
}

func messageData(headers []mimeHeader, content *mimePart, custom []mimeHeader) []byte {
	headers = applyCustomHeaders(append(slices.Clip(headers), content.headers...), custom)
	var data bytes.Buffer
	writeHeaders(&data, headers)
	data.WriteString("\r\n")
	data.Write(content.body)
	return data.Bytes()
}

func has8Bit(data []byte) bool {
	return slices.ContainsFunc(data, func(c byte) bool { return c >= 0x80 })
}

func nonEmpty(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

// content builds the message body: the text or HTML body as a single part with the message
// contentType, both as a multipart/alternative, and with attachments a multipart/mixed after it.
func (b messageBuilder) content(email *values.Map) (*mimePart, *values.Error) {
	body, _ := stringField(email, "body")
	htmlBody, _ := stringField(email, "htmlBody")
	contentType, _ := stringField(email, "contentType")
	var first *mimePart
	var err *values.Error
	switch {
	case body != "" && htmlBody == "":
		first, err = textPart(body, contentType, "text/plain")
	case body == "" && htmlBody != "":
		first, err = textPart(htmlBody, contentType, "text/html")
	case body != "" && htmlBody != "":
		first, err = alternative(body, htmlBody, contentType)
	default:
		first, err = textPart("\r\n", contentType, "text/plain")
	}
	attachments, hasAttachments := email.Get("attachments")
	if err != nil || !hasAttachments {
		return first, err
	}
	parts := []*mimePart{first}
	for _, attachment := range attachmentValues(attachments) {
		part, err := b.attachmentPart(attachment)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return multipartPart("multipart/mixed", nil, parts), nil
}

// alternative applies the message contentType to the plain-text part.
func alternative(body, htmlBody, contentType string) (*mimePart, *values.Error) {
	text, err := textPart(body, contentType, "text/plain")
	if err != nil {
		return nil, err
	}
	html, err := textPart(htmlBody, "", "text/html")
	if err != nil {
		return nil, err
	}
	return multipartPart("multipart/alternative", nil, []*mimePart{text, html}), nil
}

func attachmentValues(v values.BalValue) []values.BalValue {
	list, ok := v.(*values.List)
	if !ok {
		return []values.BalValue{v}
	}
	out := make([]values.BalValue, list.Len())
	for i := range out {
		out[i] = list.Get(i)
	}
	return out
}

func (b messageBuilder) attachmentPart(v values.BalValue) (*mimePart, *values.Error) {
	if entity, ok := v.(*values.Object); ok {
		return b.entityPart(entity)
	}
	attachment, _ := v.(*values.Map)
	return b.filePart(attachment)
}

// filePart attaches a file with the record's contentType, which cannot be a multipart type.
func (b messageBuilder) filePart(attachment *values.Map) (*mimePart, *values.Error) {
	path, _ := stringField(attachment, "filePath")
	contentType, _ := stringField(attachment, "contentType")
	mediaType, params, parseErr := parseContentType(contentType)
	if parseErr != nil {
		return nil, emailError("Invalid content type for the attachment '" + path + "': " + contentType)
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		return nil, emailError("The content type of a file attachment cannot be multipart: " + contentType)
	}
	data, read := b.files[attachment]
	if !read {
		var err error
		if data, err = b.rt.Platform().FS.ReadFile(path); err != nil {
			return nil, emailErrorWithCause(attachmentReadError, err)
		}
		b.files[attachment] = data
	}
	name := filepath.Base(path)
	cte, encoded := transferEncode(data, isTextBased(mediaType) && asciiCompatible(params["charset"]))
	params["name"] = name
	return &mimePart{
		headers: []mimeHeader{
			{"Content-Type", mime.FormatMediaType(mediaType, params)},
			{"Content-Transfer-Encoding", cte},
			{"Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name})},
		},
		body: encoded,
	}, nil
}

// entityPart converts a mime:Entity with its subtype, parameters and every header value. A
// multipart entity's body is written unencoded, so its Content-Transfer-Encoding is dropped.
func (b messageBuilder) entityPart(entity *values.Object) (*mimePart, *values.Error) {
	var headers []mimeHeader
	contentType, transferEncoding := "", ""
	for _, h := range mimenative.EntityHeaders(entity) {
		switch strings.ToLower(h.Name) {
		case "content-type":
			contentType = h.Values[0]
			continue
		case "content-transfer-encoding":
			transferEncoding = strings.TrimSpace(h.Values[0])
			continue
		}
		if !validHeaderName(h.Name) {
			return nil, emailError("Invalid header name: " + h.Name)
		}
		for _, value := range h.Values {
			if isUTF8MessageID(h.Name, value) {
				*b.utf8MessageID = true
			}
			headers = append(headers, mimeHeader{h.Name, headerValue(h.Name, value)})
		}
	}
	if children, ok := mimenative.EntityBodyParts(entity); ok {
		if contentType == "" {
			contentType = "multipart/mixed"
		}
		mediaType, params, parseErr := parseContentType(contentType)
		if parseErr != nil || !strings.HasPrefix(mediaType, "multipart/") {
			return nil, emailError("Invalid content type for the attachment: " + contentType)
		}
		if len(children) == 0 {
			return nil, emailError("A multipart attachment needs at least one body part")
		}
		if b.nesting[entity] {
			return nil, emailError("A multipart attachment contains itself")
		}
		b.nesting[entity] = true
		defer delete(b.nesting, entity)
		parts := make([]*mimePart, len(children))
		for i, child := range children {
			part, err := b.entityPart(child)
			if err != nil {
				return nil, err
			}
			parts[i] = part
		}
		part := multipartPart(mediaType, params, parts)
		part.headers = append(part.headers, headers...)
		return part, nil
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	data, err := mimenative.EntityBytes(b.ctx, entity)
	if err != nil {
		return nil, emailErrorWithCause(attachmentReadError, err)
	}
	mediaType, params, parseErr := mime.ParseMediaType(contentType)
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/") && (parseErr != nil || !validBoundary(params["boundary"])) {
		return nil, emailError("Invalid content type for the attachment: " + contentType)
	}
	text := parseErr == nil && isTextBased(mediaType) && asciiCompatible(params["charset"])
	if parseErr == nil && strings.HasPrefix(mediaType, "text/") && params["charset"] == "" {
		params["charset"] = defaultCharset(data)
		contentType = mime.FormatMediaType(mediaType, params)
	}
	var body []byte
	switch cte := strings.ToLower(transferEncoding); cte {
	case "base64":
		body = encodeBase64(data)
	case "quoted-printable":
		body = encodeQuotedPrintable(data, !text)
	case "7bit", "8bit", "binary":
		check := cte
		if b.sevenBit {
			check = "7bit"
		}
		if lineSafe(data, text, check) {
			body = toCRLF(data)
			// DATA cannot carry the binary encoding (RFC 3030), nor 8bit without 8BITMIME, so
			// line-safe data is relabelled.
			if cte != check || cte == "binary" {
				transferEncoding = "7bit"
				if has8Bit(data) {
					transferEncoding = "8bit"
				}
			}
		} else {
			transferEncoding, body = transferEncode(data, text)
		}
	case "":
		transferEncoding, body = transferEncode(data, text)
	default:
		return nil, emailError("Unsupported Content-Transfer-Encoding: " + transferEncoding)
	}
	part := &mimePart{
		headers: []mimeHeader{
			{"Content-Type", headerValue("Content-Type", contentType)},
			{"Content-Transfer-Encoding", transferEncoding},
		},
		body: body,
	}
	part.headers = append(part.headers, headers...)
	return part, nil
}

// lineSafe reports whether data can be sent unencoded under an identity transfer encoding: line
// endings may be canonicalized only for text, and 7bit data must be ASCII.
func lineSafe(data []byte, text bool, cte string) bool {
	if hasLongLine(data) || bytes.IndexByte(data, 0) >= 0 || (!text && hasBareLineBreak(data)) {
		return false
	}
	_, nonASCII := asciiCounts(data)
	return cte != "7bit" || nonASCII == 0
}

func defaultCharset(data []byte) string {
	if _, nonASCII := asciiCounts(data); nonASCII > 0 {
		return "UTF-8"
	}
	return "us-ascii"
}

// textPart encodes text in the content type's charset, adding a charset parameter to a text type
// that has none, as JavaMail does.
func textPart(text, contentType, defaultType string) (*mimePart, *values.Error) {
	if contentType == "" {
		contentType = defaultType
	}
	mediaType, params, parseErr := parseContentType(contentType)
	if parseErr != nil {
		return nil, emailError("Invalid content type: " + contentType)
	}
	charset := params["charset"]
	data := []byte(text)
	switch {
	case charset != "":
		encoded, err := mimenative.EncodeCharset(text, charset)
		if err != nil {
			return nil, emailError("Unsupported charset: " + charset)
		}
		data = encoded
	case strings.HasPrefix(mediaType, "text/"):
		params["charset"] = defaultCharset(data)
		contentType = mime.FormatMediaType(mediaType, params)
	}
	cte, encoded := transferEncode(data, asciiCompatible(charset))
	return &mimePart{
		headers: []mimeHeader{{"Content-Type", headerValue("Content-Type", contentType)}, {"Content-Transfer-Encoding", cte}},
		body:    encoded,
	}, nil
}

// asciiCompatible reports whether charset encodes CR and LF as ASCII does; the output of one that
// does not, such as UTF-16, is transferred as binary data so that no line ending is rewritten.
func asciiCompatible(charset string) bool {
	if charset == "" {
		return true
	}
	encoded, err := mimenative.EncodeCharset("\r\n", charset)
	return err == nil && string(encoded) == "\r\n"
}

// parseContentType parses a Content-Type value, which unlike a Content-Disposition value needs a
// type/subtype pair.
func parseContentType(contentType string) (string, map[string]string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && !strings.Contains(mediaType, "/") {
		err = errors.New("missing subtype")
	}
	return mediaType, params, err
}

func multipartPart(mediaType string, params map[string]string, parts []*mimePart) *mimePart {
	if params == nil {
		params = map[string]string{}
	}
	boundary := params["boundary"]
	if !validBoundary(boundary) || slices.ContainsFunc(parts, func(p *mimePart) bool { return bytes.Contains(p.body, []byte("--"+boundary)) }) {
		boundary = "----=_Part_" + randomHex(12)
		params["boundary"] = boundary
	}
	var body bytes.Buffer
	for _, part := range parts {
		body.WriteString("--" + boundary + "\r\n")
		writeHeaders(&body, part.headers)
		body.WriteString("\r\n")
		body.Write(part.body)
		// The CRLF before the next boundary belongs to the delimiter (RFC 2046), not to the body.
		body.WriteString("\r\n")
	}
	body.WriteString("--" + boundary + "--\r\n")
	part := &mimePart{
		headers: []mimeHeader{{"Content-Type", mime.FormatMediaType(mediaType, params)}},
		body:    body.Bytes(),
	}
	// A multipart body holding 8-bit data must not keep the default 7bit label (RFC 2045 section 6.4).
	if has8Bit(part.body) {
		part.headers = append(part.headers, mimeHeader{"Content-Transfer-Encoding", "8bit"})
	}
	return part
}

// validBoundary reports whether boundary is 1 to 70 RFC 2046 bchars that do not end in a space.
func validBoundary(boundary string) bool {
	if boundary == "" || len(boundary) > 70 || strings.HasSuffix(boundary, " ") {
		return false
	}
	for _, c := range []byte(boundary) {
		if !isBoundaryChar(c) {
			return false
		}
	}
	return true
}

func isBoundaryChar(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || strings.IndexByte("'()+_,-./:=? ", c) >= 0
}

// customHeaders reads Message.headers, rejecting names that are not RFC 5322 field names.
func customHeaders(email *values.Map) ([]mimeHeader, *values.Error) {
	v, ok := email.Get("headers")
	if !ok {
		return nil, nil
	}
	custom, _ := v.(*values.Map)
	headers := make([]mimeHeader, 0, custom.Len())
	for _, name := range custom.Keys() {
		if !validHeaderName(name) {
			return nil, emailError("Invalid header name: " + name)
		}
		value, _ := stringField(custom, name)
		headers = append(headers, mimeHeader{name, value})
	}
	return headers, nil
}

// addressHeader returns the addresses for an address header: a custom header of that name replaces
// the Message field, so the envelope always matches the written header, as with JavaMail.
func addressHeader(custom []mimeHeader, name string, fieldValues []string) ([]*mail.Address, *values.Error) {
	for _, h := range custom {
		if strings.EqualFold(h.name, name) {
			return parseAddressList(h.value)
		}
	}
	return parseAddresses(fieldValues)
}

// structuralHeaders are generated from the message content; like JavaMail, the generated values win.
var structuralHeaders = []string{"mime-version", "content-type", "content-transfer-encoding"}

// applyCustomHeaders sets each remaining Message.headers entry, replacing a header of the same
// name. Address headers were already applied by addressHeader, and Bcc is never written.
func applyCustomHeaders(headers []mimeHeader, custom []mimeHeader) []mimeHeader {
	for _, h := range custom {
		lower := strings.ToLower(h.name)
		if slices.Contains(structuralHeaders, lower) || slices.Contains(addressHeaders, lower) {
			continue
		}
		value := headerValue(h.name, h.value)
		replaced := false
		for i := range headers {
			if strings.EqualFold(headers[i].name, h.name) {
				headers[i].value = value
				replaced = true
			}
		}
		if !replaced {
			headers = append(headers, mimeHeader{h.name, value})
		}
	}
	return headers
}

var addressHeaders = []string{"from", "sender", "reply-to", "to", "cc", "bcc"}

var messageIDHeaders = []string{"message-id", "in-reply-to", "references", "content-id"}

// isUTF8MessageID reports whether a header is a non-ASCII message ID header, which needs SMTPUTF8.
func isUTF8MessageID(name, value string) bool {
	_, nonASCII := asciiCounts([]byte(value))
	return nonASCII > 0 && slices.Contains(messageIDHeaders, strings.ToLower(name))
}

// headerValue prepares a header value by kind: MIME parameter headers get RFC 2231 parameters,
// Subject and Comments RFC 2047 encoded-words, and other headers encoded-words only for non-ASCII.
func headerValue(name, value string) string {
	switch strings.ToLower(name) {
	case "content-type", "content-disposition":
		if _, nonASCII := asciiCounts([]byte(value)); nonASCII > 0 {
			if mediaType, params, err := mime.ParseMediaType(value); err == nil {
				return mime.FormatMediaType(mediaType, params)
			}
		}
		return value
	case "subject", "comments":
		return encodeHeaderValue(value)
	}
	// Message IDs keep their syntax; a non-ASCII one is sent as UTF-8 under SMTPUTF8 (RFC 6532).
	if slices.Contains(messageIDHeaders, strings.ToLower(name)) {
		return value
	}
	// ASCII values are kept, since encoded-words are not allowed in structured fields (RFC 2047 section 5).
	if _, nonASCII := asciiCounts([]byte(value)); nonASCII == 0 {
		return value
	}
	return encodeHeaderValue(value)
}

// writeHeaders unfolds line breaks in values, so a value cannot start a new header, and then folds
// long lines at spaces.
func writeHeaders(buf *bytes.Buffer, headers []mimeHeader) {
	for _, h := range headers {
		buf.WriteString(foldHeader(h.name + ": " + headerLineBreaks.Replace(h.value)))
		buf.WriteString("\r\n")
	}
}

// foldHeader breaks a header line at spaces so lines stay within 76 columns where possible, as
// JavaMail's MimeUtility.fold does.
func foldHeader(line string) string {
	if len(line) <= foldColumn {
		return line
	}
	var b strings.Builder
	column := 0
	for i, word := range strings.Split(line, " ") {
		if i > 0 {
			if column > 1 && column+1+len(word) > foldColumn {
				b.WriteString("\r\n")
				column = 0
			}
			b.WriteByte(' ')
			column++
		}
		b.WriteString(word)
		column += len(word)
	}
	return b.String()
}

var headerLineBreaks = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ")

// validHeaderName reports whether name is an RFC 5322 field name: printable ASCII without colon.
func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range []byte(name) {
		if c <= ' ' || c >= 0x7f || c == ':' {
			return false
		}
	}
	return true
}

// parseAddress accepts RFC 5322 addresses and, like JavaMail, a bare local part without a domain.
func parseAddress(value string) (*mail.Address, *values.Error) {
	if addr, err := mail.ParseAddress(value); err == nil {
		return addr, nil
	}
	local := strings.TrimSpace(value)
	if local == "" || strings.ContainsAny(local, ",;<>()[]:\"") {
		return nil, emailError("Illegal address")
	}
	name, _, hasDomain := strings.Cut(local, "@")
	if err := checkLocalPart(name, hasDomain); err != nil {
		return nil, err
	}
	if hasDomain {
		return nil, emailError("Illegal address")
	}
	return &mail.Address{Address: local}, nil
}

// checkLocalPart reports the first character JavaMail rejects in an unquoted local part; a
// trailing dot is rejected only before a domain.
func checkLocalPart(local string, hasDomain bool) *values.Error {
	for i, r := range local {
		switch {
		case unicode.IsSpace(r) || unicode.IsControl(r):
			return emailError("Local address contains control or whitespace")
		case r == '.' && i == 0:
			return emailError("Local address starts with dot")
		case r == '.' && local[i-1] == '.':
			return emailError("Local address contains dot-dot")
		}
	}
	if hasDomain && strings.HasSuffix(local, ".") {
		return emailError("Local address ends with dot")
	}
	return nil
}

// parseAddressList parses a comma-separated header value, falling back to splitting at commas so
// bare local parts are accepted as parseAddress accepts them. An empty value is an empty list.
func parseAddressList(value string) ([]*mail.Address, *values.Error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	if list, err := mail.ParseAddressList(value); err == nil {
		return list, nil
	}
	return parseAddresses(splitAddressList(value))
}

// splitAddressList splits a header value at the commas outside quoted strings, comments and
// angle brackets.
func splitAddressList(value string) []string {
	var parts []string
	start, depth, quoted, angle := 0, 0, false, false
	for i := 0; i < len(value); i++ {
		switch c := value[i]; {
		case (quoted || depth > 0) && c == '\\':
			i++
		case c == '"' && depth == 0:
			quoted = !quoted
		case quoted:
		case c == '(':
			depth++
		case c == ')' && depth > 0:
			depth--
		case depth > 0:
		case c == '<':
			angle = true
		case c == '>':
			angle = false
		case c == ',' && !angle:
			parts = append(parts, value[start:i])
			start = i + 1
		}
	}
	return append(parts, value[start:])
}

func parseAddresses(addresses []string) ([]*mail.Address, *values.Error) {
	out := make([]*mail.Address, 0, len(addresses))
	for _, value := range addresses {
		addr, err := parseAddress(value)
		if err != nil {
			return nil, err
		}
		out = append(out, addr)
	}
	return out, nil
}

// formatAddresses writes an address without a display name bare, without angle brackets, as
// JavaMail does.
func formatAddresses(addrs []*mail.Address) string {
	out := make([]string, len(addrs))
	for i, addr := range addrs {
		if addr.Name == "" {
			out[i] = addrSpec(addr)
		} else {
			out[i] = addr.String()
		}
	}
	return strings.Join(out, ", ")
}

// addrSpec renders an address for the envelope and headers, re-quoting a local part that is not
// a dot-atom; net/mail stores it unquoted, which would let it break the SMTP command syntax.
func addrSpec(addr *mail.Address) string {
	if !strings.Contains(addr.Address, "@") {
		return addr.Address
	}
	return strings.TrimSuffix(strings.TrimPrefix((&mail.Address{Address: addr.Address}).String(), "<"), ">")
}

// encodeHeaderValue RFC 2047-encodes a non-ASCII value, with Q encoding when it is mostly ASCII
// and B encoding otherwise, as JavaMail does.
func encodeHeaderValue(value string) string {
	ascii, nonASCII := asciiCounts([]byte(value))
	switch {
	case nonASCII == 0 && !hasLongWord(value):
		return value
	case nonASCII == 0:
		return encodeWords(value)
	case ascii > nonASCII:
		return mime.QEncoding.Encode("UTF-8", value)
	default:
		return mime.BEncoding.Encode("UTF-8", value)
	}
}

// hasLongWord reports whether value has a run without spaces too long for a folded header line.
func hasLongWord(value string) bool {
	return slices.ContainsFunc(strings.Split(value, " "), func(word string) bool { return len(word) > maxLineLength-foldColumn })
}

// encodeWords B-encodes value as short encoded-words, which folding can then split onto separate lines.
func encodeWords(value string) string {
	var words []string
	for len(value) > 0 {
		n := min(len(value), 45)
		for n < len(value) && !utf8.RuneStart(value[n]) {
			n--
		}
		words = append(words, "=?UTF-8?B?"+base64.StdEncoding.EncodeToString([]byte(value[:n]))+"?=")
		value = value[n:]
	}
	return strings.Join(words, " ")
}

func messageID(from string) string {
	domain := "localhost"
	if i := strings.LastIndex(from, "@"); i >= 0 && i < len(from)-1 {
		domain = from[i+1:]
	}
	return "<" + randomHex(16) + "@" + domain + ">"
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// isTextBased mirrors jBallerina's text detection for attachment bodies.
func isTextBased(mediaType string) bool {
	return strings.HasPrefix(mediaType, "text/") || strings.HasSuffix(mediaType, "+xml") ||
		strings.HasSuffix(mediaType, "+json") || mediaType == "application/json" ||
		mediaType == "application/xml" || mediaType == "application/x-www-form-urlencoded"
}

// asciiCounts classifies bytes as JavaMail does: control characters other than CR, LF and TAB
// count as non-ASCII.
func asciiCounts(data []byte) (ascii, nonASCII int) {
	for _, c := range data {
		if c >= 0x7f || (c < 0x20 && c != '\r' && c != '\n' && c != '\t') {
			nonASCII++
		} else {
			ascii++
		}
	}
	return ascii, nonASCII
}

func hasLongLine(data []byte) bool {
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		if len(bytes.TrimSuffix(line, []byte("\r"))) > maxLineLength {
			return true
		}
	}
	return false
}

// transferEncode picks the Content-Transfer-Encoding as JavaMail does: 7bit for short-lined
// ASCII, quoted-printable for text with more ASCII than non-ASCII bytes, and base64 otherwise.
// Unlike JavaMail, non-text data with a bare CR or LF is not 7bit, as sending would change it.
func transferEncode(data []byte, text bool) (string, []byte) {
	ascii, nonASCII := asciiCounts(data)
	switch {
	case nonASCII == 0 && !hasLongLine(data) && (text || !hasBareLineBreak(data)):
		return "7bit", toCRLF(data)
	case text && ascii > nonASCII:
		return "quoted-printable", encodeQuotedPrintable(data, false)
	default:
		return "base64", encodeBase64(data)
	}
}

// encodeQuotedPrintable encodes data, keeping CR and LF bytes as they are when binary is set.
func encodeQuotedPrintable(data []byte, binary bool) []byte {
	var buf bytes.Buffer
	w := quotedprintable.NewWriter(&buf)
	w.Binary = binary
	_, _ = w.Write(data)
	_ = w.Close()
	return toCRLF(buf.Bytes())
}

func encodeBase64(data []byte) []byte {
	encoded := mimenative.MimeBase64(data)
	out := make([]byte, len(encoded)+2)
	copy(out, encoded)
	copy(out[len(encoded):], "\r\n")
	return out
}

// toCRLF normalizes bare LF and bare CR line endings to CRLF, returning data itself when it has none.
func toCRLF(data []byte) []byte {
	if !hasBareLineBreak(data) {
		return data
	}
	out := make([]byte, 0, len(data)+len(data)/32+2)
	for i := 0; i < len(data); i++ {
		switch c := data[i]; c {
		case '\r':
			out = append(out, '\r', '\n')
			if i+1 < len(data) && data[i+1] == '\n' {
				i++
			}
		case '\n':
			out = append(out, '\r', '\n')
		default:
			out = append(out, c)
		}
	}
	return out
}

func hasBareLineBreak(data []byte) bool {
	for i, c := range data {
		if (c == '\r' && (i+1 == len(data) || data[i+1] != '\n')) || (c == '\n' && (i == 0 || data[i-1] != '\r')) {
			return true
		}
	}
	return false
}
