// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"slices"
	"strings"
	"unicode/utf8"

	mimenative "github.com/ballerina-nutcracker/ballerina/lib/stdlibs/ballerina/mime/0.0.1/go1.27/native"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type rawHeader struct {
	name, value string
}

// mimeNode is a parsed MIME entity: a leaf with its transfer-decoded body, or a multipart
// entity with its children.
type mimeNode struct {
	headers   []rawHeader
	mediaType string
	params    map[string]string
	body      []byte
	children  []*mimeNode
	multipart bool
}

// receivedContent is the message content split into the email:Message body fields.
type receivedContent struct {
	body        *string
	htmlBody    *string
	attachments []mimenative.Part
}

var errMalformedMultipart = errors.New("missing start boundary")

// maxMultipartDepth bounds multipart nesting, as each level rescans the bytes of the levels inside it.
const maxMultipartDepth = 32

var errMultipartDepth = errors.New("multipart nesting too deep")

// convertReceived builds an email:Message from a raw RFC 5322 message.
func convertReceived(ctx *extern.Context, strArrTy semtypes.SemType, data []byte) (values.BalValue, error) {
	headers, rawBody := splitEntity(data)
	content := receivedContent{}
	if node, err := parseNode(headers, rawBody, 0); err == nil {
		content.collectTop(node)
	} else {
		// A message whose structure cannot be parsed is delivered with its raw body.
		raw, _ := mimenative.DecodeCharset(rawBody, "")
		content.body = &raw
	}
	attachments, err := mimenative.NewEntities(ctx, content.attachments)
	if err != nil {
		return nil, err
	}
	contentType := headerValueOf(headers, "Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}
	headerEntries := make([]string, 0, 2*len(headers))
	for _, h := range headers {
		headerEntries = append(headerEntries, h.name, h.value)
	}
	replyTo := receivedAddresses(headers, "Reply-To")
	if replyTo == nil {
		replyTo = receivedAddresses(headers, "From")
	}
	body := ""
	if content.body != nil {
		body = *content.body
	}
	var htmlBody values.BalValue
	if content.htmlBody != nil {
		htmlBody = *content.htmlBody
	}
	atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), strArrTy)
	list := func(items []string) *values.List {
		vals := make([]values.BalValue, len(items))
		for i, s := range items {
			vals[i] = s
		}
		return values.NewList(strArrTy, atomic, false, nil, len(vals), vals)
	}
	handle, ok := ctx.LookupFunction(orgName, moduleName, "newReceivedMessage")
	if !ok {
		return nil, errors.New("email: internal helper function newReceivedMessage not found")
	}
	return ctx.InvokeFunction(handle, []values.BalValue{
		list(receivedAddresses(headers, "To")),
		list(receivedAddresses(headers, "Cc")),
		list(receivedAddresses(headers, "Bcc")),
		list(replyTo),
		decodeHeaderText(headerValueOf(headers, "Subject")),
		body,
		htmlBody,
		contentType,
		list(headerEntries),
		firstAddress(headers, "From"),
		firstAddress(headers, "Sender"),
		attachments,
	})
}

// splitEntity splits an entity into its unfolded headers, in order, and its body. A header line
// that is not UTF-8 is read as ISO-8859-1, as JavaMail reads every header line.
func splitEntity(data []byte) ([]rawHeader, []byte) {
	var headers []rawHeader
	// folded holds the continuation lines of the last header, joined once it ends.
	var folded []string
	unfold := func() {
		if len(folded) > 0 {
			headers[len(headers)-1].value += strings.Join(folded, "")
			folded = folded[:0]
		}
	}
	rest := data
	for len(rest) > 0 {
		raw, next, _ := bytes.Cut(rest, []byte("\n"))
		raw = bytes.TrimSuffix(raw, []byte("\r"))
		if len(raw) == 0 {
			unfold()
			return headers, next
		}
		rest = next
		line := string(raw)
		if !utf8.ValidString(line) {
			line, _ = mimenative.DecodeCharset(raw, "ISO-8859-1")
		}
		if (line[0] == ' ' || line[0] == '\t') && len(headers) > 0 {
			folded = append(folded, line)
			continue
		}
		unfold()
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		headers = append(headers, rawHeader{strings.TrimSpace(name), strings.TrimLeft(value, " \t")})
	}
	unfold()
	return headers, nil
}

func headerValueOf(headers []rawHeader, name string) string {
	for _, h := range headers {
		if strings.EqualFold(h.name, name) {
			return strings.TrimSpace(h.value)
		}
	}
	return ""
}

// parseNode parses an entity's content type and body, recursing into multipart bodies. Only a
// broken top-level structure is an error; a part with an unparseable type, or a multipart part
// nested deeper than maxMultipartDepth, is kept as binary data.
func parseNode(headers []rawHeader, body []byte, depth int) (*mimeNode, error) {
	node := &mimeNode{headers: headers, mediaType: "text/plain", params: map[string]string{}}
	if contentType := headerValueOf(headers, "Content-Type"); contentType != "" {
		mediaType, params, err := parseMediaType(contentType)
		if err != nil {
			return nil, err
		}
		node.mediaType, node.params = mediaType, params
	}
	if !strings.HasPrefix(node.mediaType, "multipart/") {
		node.body = transferDecode(body, headerValueOf(headers, "Content-Transfer-Encoding"))
		return node, nil
	}
	if depth >= maxMultipartDepth {
		return nil, errMultipartDepth
	}
	parts, err := splitMultipart(body, node.params["boundary"])
	if err != nil {
		return nil, err
	}
	node.multipart = true
	for _, part := range parts {
		partHeaders, partBody := splitEntity(part)
		child, err := parseNode(partHeaders, partBody, depth+1)
		if err != nil {
			child = binaryNode(partHeaders, partBody)
		}
		node.children = append(node.children, child)
	}
	return node, nil
}

// binaryNode keeps a part whose structure cannot be parsed as application/octet-stream data.
func binaryNode(headers []rawHeader, body []byte) *mimeNode {
	kept := make([]rawHeader, 0, len(headers))
	for _, h := range headers {
		if strings.EqualFold(h.name, "Content-Type") {
			h.value = "application/octet-stream"
		}
		kept = append(kept, h)
	}
	return &mimeNode{headers: kept, mediaType: "application/octet-stream", params: map[string]string{},
		body: transferDecode(body, headerValueOf(headers, "Content-Transfer-Encoding"))}
}

// parseMediaType is mime.ParseMediaType, except that a malformed parameter drops the parameters
// but keeps the media type.
func parseMediaType(value string) (string, map[string]string, error) {
	mediaType, params, err := mime.ParseMediaType(value)
	if errors.Is(err, mime.ErrInvalidMediaParameter) {
		return mediaType, map[string]string{}, nil
	}
	return mediaType, params, err
}

// splitMultipart returns the raw parts between the boundary delimiters (RFC 2046 section 5.1.1).
// A missing close delimiter is tolerated, as JavaMail does.
func splitMultipart(body []byte, boundary string) ([][]byte, error) {
	if boundary == "" {
		return nil, errMalformedMultipart
	}
	delimiter := []byte("--" + boundary)
	var parts [][]byte
	start := -1
	for offset := 0; offset < len(body); {
		end := len(body)
		if i := bytes.IndexByte(body[offset:], '\n'); i >= 0 {
			end = offset + i + 1
		}
		line := bytes.TrimRight(body[offset:end], " \t\r\n")
		if rest, ok := bytes.CutPrefix(line, delimiter); ok && (len(rest) == 0 || bytes.Equal(rest, []byte("--"))) {
			if start >= 0 {
				parts = append(parts, trimLineBreak(body[start:offset]))
			}
			if len(rest) > 0 {
				return parts, nil
			}
			start = end
		}
		offset = end
	}
	if start < 0 {
		return nil, errMalformedMultipart
	}
	if start < len(body) {
		parts = append(parts, trimLineBreak(body[start:]))
	}
	return parts, nil
}

// trimLineBreak drops the line break that belongs to the following delimiter.
func trimLineBreak(part []byte) []byte {
	part = bytes.TrimSuffix(part, []byte("\n"))
	return bytes.TrimSuffix(part, []byte("\r"))
}

// transferDecode decodes a body per its Content-Transfer-Encoding, keeping the raw bytes when
// they do not decode. Base64 padding is optional, as in JavaMail.
func transferDecode(body []byte, encoding string) []byte {
	var decoded []byte
	var err error
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		data := bytes.TrimRight(bytes.Map(func(c rune) rune {
			if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
				return -1
			}
			return c
		}, body), "=")
		decoded = make([]byte, base64.RawStdEncoding.DecodedLen(len(data)))
		var n int
		n, err = base64.RawStdEncoding.Decode(decoded, data)
		decoded = decoded[:n]
	case "quoted-printable":
		decoded, err = io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
	default:
		return body
	}
	if err != nil {
		return body
	}
	return decoded
}

// collectTop splits the top-level entity: a single text part is the body (HTML the htmlBody),
// a single non-text part or one marked as an attachment is an attachment, and a multipart entity
// is searched for its body parts.
func (c *receivedContent) collectTop(node *mimeNode) {
	switch {
	case isAttachment(node), !node.multipart && !isTextBased(node.mediaType):
		c.attachments = append(c.attachments, contentPart(node))
	case node.multipart, node.mediaType == "text/html":
		c.collect(node)
	default:
		text := nodeText(node)
		c.body = &text
	}
}

// collect fills body and htmlBody from a part that wants reports, entering only the multipart
// parts that hold one; every other part is an attachment.
func (c *receivedContent) collect(node *mimeNode) {
	switch {
	case node.multipart:
		for _, child := range node.children {
			if c.wants(child) {
				c.collect(child)
			} else {
				c.attachments = append(c.attachments, contentPart(child))
			}
		}
	case node.mediaType == "text/html":
		html := nodeText(node)
		c.htmlBody = &html
	default:
		text := nodeText(node)
		c.body = &text
	}
}

// wants reports whether node is, or holds outside attachments, a body part still unfilled.
func (c *receivedContent) wants(node *mimeNode) bool {
	switch {
	case isAttachment(node):
		return false
	case node.multipart:
		return slices.ContainsFunc(node.children, c.wants)
	case node.mediaType == "text/plain":
		return c.body == nil
	case node.mediaType == "text/html":
		return c.htmlBody == nil
	default:
		return false
	}
}

func isAttachment(node *mimeNode) bool {
	disposition, _, err := parseMediaType(headerValueOf(node.headers, "Content-Disposition"))
	return err == nil && disposition == "attachment"
}

// nodeText decodes a text body in its charset, falling back to UTF-8 for an unknown charset.
func nodeText(node *mimeNode) string {
	text, err := mimenative.DecodeCharset(node.body, node.params["charset"])
	if err != nil {
		text, _ = mimenative.DecodeCharset(node.body, "")
	}
	return text
}

// contentPart converts a node to an attachment with its content headers; a leaf without a
// Content-Type gets the RFC 2045 default.
func contentPart(node *mimeNode) mimenative.Part {
	part := mimenative.Part{Body: node.body, Multipart: node.multipart}
	if !node.multipart && headerValueOf(node.headers, "Content-Type") == "" {
		part.Headers = append(part.Headers, mimenative.EntityHeader{Name: "Content-Type", Values: []string{"text/plain"}})
	}
	for _, h := range node.headers {
		if strings.HasPrefix(strings.ToLower(h.name), "content-") {
			part.Headers = append(part.Headers, mimenative.EntityHeader{Name: h.name, Values: []string{h.value}})
		}
	}
	for _, child := range node.children {
		part.Parts = append(part.Parts, contentPart(child))
	}
	return part
}

// receivedAddresses returns the addresses of every header with the given name as JavaMail
// formats them, or nil when there is no such header. Unparseable values are returned as written.
func receivedAddresses(headers []rawHeader, name string) []string {
	var raw []string
	for _, h := range headers {
		if strings.EqualFold(h.name, name) && strings.TrimSpace(h.value) != "" {
			raw = append(raw, h.value)
		}
	}
	if raw == nil {
		return nil
	}
	parser := &mail.AddressParser{WordDecoder: headerWordDecoder}
	list, err := parser.ParseList(strings.Join(raw, ", "))
	if err != nil {
		var out []string
		for _, value := range raw {
			for _, item := range splitAddressList(value) {
				if item = strings.TrimSpace(item); item == "" {
					continue
				}
				if addr, err := parser.Parse(item); err == nil {
					out = append(out, displayAddress(addr))
				} else {
					out = append(out, item)
				}
			}
		}
		return out
	}
	out := make([]string, len(list))
	for i, addr := range list {
		out[i] = displayAddress(addr)
	}
	return out
}

func firstAddress(headers []rawHeader, name string) string {
	if list := receivedAddresses(headers, name); len(list) > 0 {
		return list[0]
	}
	return ""
}

// displayAddress formats an address as JavaMail's InternetAddress.toString does: the display name
// is quoted only when it needs to be, and encoded when it is not ASCII.
func displayAddress(addr *mail.Address) string {
	spec := addrSpec(addr)
	if addr.Name == "" {
		return spec
	}
	name := addr.Name
	if _, nonASCII := asciiCounts([]byte(name)); nonASCII > 0 {
		name = encodeHeaderValue(name)
	} else if strings.ContainsAny(name, "()<>@,;:\\\".[]") {
		name = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name) + `"`
	}
	return name + " <" + spec + ">"
}

// headerWordDecoder decodes RFC 2047 encoded-words in any charset the mime module supports.
var headerWordDecoder = &mime.WordDecoder{CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	text, err := mimenative.DecodeCharset(data, charset)
	if err != nil {
		return nil, err
	}
	return strings.NewReader(text), nil
}}

func decodeHeaderText(value string) string {
	decoded, err := headerWordDecoder.DecodeHeader(value)
	if err != nil {
		return value
	}
	return decoded
}
