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

package extern_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
)

// emailPal wraps the default TestPal with the real platform dialer and TLS upgrade, so the email
// clients under test talk to the in-process fake mail servers.
type emailPal struct {
	testharness.TestPal
	// refused is an address whose dial fails with ECONNREFUSED without a connection attempt.
	refused string
	// readOnce is a file that can be read only once; later reads fail.
	readOnce string
	reads    atomic.Int32
}

func newEmailPal() *emailPal {
	return &emailPal{TestPal: testharness.NewTestPal()}
}

func (p *emailPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.Net = pal.Net{Dial: func(ctx context.Context, network, address, localAddr string, tlsCfg *pal.TLSConfig) (net.Conn, error) {
		if address == p.refused {
			return nil, &net.OpError{Op: "dial", Net: network, Err: syscall.ECONNREFUSED}
		}
		return palnative.Dial(ctx, network, address, localAddr, tlsCfg)
	}, UpgradeTLS: palnative.UpgradeTLS}
	readFile := base.FS.ReadFile
	base.FS.ReadFile = func(path string) ([]byte, error) {
		if path == p.readOnce && p.reads.Add(1) > 1 {
			return nil, errors.New("file already read")
		}
		return readFile(path)
	}
	return base
}

func runEmailCase(t *testing.T, name, balContent string, externs ...testharness.ExternRegistration) {
	t.Helper()
	runEmailCaseWith(t, newEmailPal(), name, balContent, externs...)
}

func runEmailCaseWith(t *testing.T, p *emailPal, name, balContent string, externs ...testharness.ExternRegistration) {
	t.Helper()
	tmpBalFile := filepath.Join(t.TempDir(), name+".bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}
	tc := test_util.TestCase{
		Name:         name,
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, name+".txtar"),
	}
	runExtern(t, tc, p, externs)
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return filepath.ToSlash(path)
}

// messagesBySubject indexes received messages by their decoded Subject header.
func messagesBySubject(t *testing.T, received []*smtpReceived) map[string]*smtpReceived {
	t.Helper()
	out := map[string]*smtpReceived{}
	for _, r := range received {
		msg, err := mail.ReadMessage(strings.NewReader(r.data))
		if err != nil {
			t.Fatalf("parsing received message: %v\n%s", err, r.data)
		}
		subject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
		if err != nil {
			t.Fatalf("decoding subject: %v", err)
		}
		if _, dup := out[subject]; dup {
			t.Fatalf("subject %q received more than once", subject)
		}
		out[subject] = r
	}
	return out
}

// need returns the message with subject, failing the test when it was not received.
func need(t *testing.T, got map[string]*smtpReceived, subject string) *smtpReceived {
	t.Helper()
	r, ok := got[subject]
	if !ok {
		t.Fatalf("no message with subject %q", subject)
	}
	return r
}

// mimeStructure renders a received message's MIME tree as a compact, boundary-free string:
// multipart types list their parts in brackets, and leaf parts show their content type, transfer
// encoding, selected headers and decoded body.
func mimeStructure(t *testing.T, data string) string {
	t.Helper()
	msg, err := mail.ReadMessage(strings.NewReader(data))
	if err != nil {
		t.Fatalf("parsing message: %v", err)
	}
	body, _ := io.ReadAll(msg.Body)
	return partStructure(t, textproto.MIMEHeader(msg.Header), body, true)
}

// topLevel trims the line break that SMTP's DATA framing adds after a single-part body.
func partStructure(t *testing.T, header textproto.MIMEHeader, body []byte, topLevel bool) string {
	t.Helper()
	contentType := header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parsing content type %q: %v", contentType, err)
	}
	boundary := params["boundary"]
	delete(params, "boundary")
	typeText := mime.FormatMediaType(mediaType, params)
	var extras []string
	for _, name := range []string{"Content-Disposition", "Content-Id", "X-Tag", "X-Part"} {
		if values := header.Values(name); len(values) > 0 {
			extras = append(extras, name+"="+strings.Join(values, ","))
		}
	}
	extra := ""
	if len(extras) > 0 {
		extra = "{" + strings.Join(extras, ";") + "}"
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		var parts []string
		for {
			part, err := reader.NextRawPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("reading part: %v", err)
			}
			partBody, _ := io.ReadAll(part)
			parts = append(parts, partStructure(t, part.Header, partBody, false))
		}
		if cte := header.Get("Content-Transfer-Encoding"); cte != "" {
			typeText += "(" + cte + ")"
		}
		return typeText + extra + "[" + strings.Join(parts, " | ") + "]"
	}
	cte := strings.ToLower(header.Get("Content-Transfer-Encoding"))
	decoded := body
	switch cte {
	case "base64":
		decoded, err = base64.StdEncoding.DecodeString(strings.NewReplacer("\r", "", "\n", "").Replace(string(body)))
		if err != nil {
			t.Fatalf("decoding base64: %v", err)
		}
	case "quoted-printable":
		decoded, err = io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			t.Fatalf("decoding quoted-printable: %v", err)
		}
	}
	// Only line-oriented encodings carry line endings that SMTP may canonicalize; base64 data is exact.
	text := string(decoded)
	if cte != "base64" {
		text = strings.ReplaceAll(text, "\r\n", "\n")
		if topLevel {
			text = strings.TrimSuffix(text, "\n")
		}
	}
	return fmt.Sprintf("%s(%s)%s%q", typeText, cte, extra, text)
}

func headerValues(t *testing.T, data, name string) []string {
	t.Helper()
	msg, err := mail.ReadMessage(strings.NewReader(data))
	if err != nil {
		t.Fatalf("parsing message: %v", err)
	}
	return msg.Header[textproto.CanonicalMIMEHeaderKey(name)]
}

func TestEmailSmtpSend(t *testing.T) {
	t.Parallel()
	server := newFakeSmtpServer(t, smtpPlain, nil, nil)
	jsonFile := writeTempFile(t, "data.json", []byte(`{"k": 1}`))
	longText := strings.Repeat("x", 1200)
	longSubject := strings.Repeat("subject word ", 100)
	// Two 998-character lines are the longest that 7bit allows.
	exactLines := strings.Repeat("y", 998) + `\r\n` + strings.Repeat("z", 998)

	runEmailCase(t, "email-smtp-send-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/mime;

public function main() returns error? {
    email:SmtpClient smtp = check new ("127.0.0.1", "alice", "secret", port = %[1]s, security = email:START_TLS_NEVER);

    check smtp->send("bob@localhost", "text", "alice@localhost", "Hello\nWorld");
    check smtp->send("bob@localhost", "bare-cr", "alice@localhost", "a\rb\r\nc");
    check smtp->sendMessage({to: "bob@localhost", subject: "html", htmlBody: "<b>Hi</b>"});
    check smtp->sendMessage({to: "bob@localhost", subject: "html-as-plain", htmlBody: "<b>Hi</b>", contentType: "text/plain"});
    check smtp->sendMessage({to: "bob@localhost", subject: "alternative", 'from: "alice@localhost",
        body: "café", htmlBody: "<p>café</p>", contentType: "text/plain; charset=ISO-8859-1"});
    check smtp->sendMessage({to: "bob@localhost", subject: "empty"});
    check smtp->send("bob@localhost", "unicode ü subject", "alice@localhost", "Grüße aus Köln");
    check smtp->send("bob@localhost", "日本語の件名", "alice@localhost", "%[3]s");
    check smtp->send("bob@localhost", "injected\r\nBcc: eve@localhost", "alice@localhost", "Body",
        headers = {"X-Note": "line1\nX-Injected: yes"});
    check smtp->send("bob@localhost", "headers", "alice@localhost", "Body",
        headers = {"Subject": "headers-overridden", "From": "other@localhost", "X-Trace": "42"});

    mime:Entity note = new;
    note.setText("note");
    note.addHeader("X-Tag", "first");
    note.addHeader("X-Tag", "second");

    mime:Entity image = new;
    image.setByteArray([137, 80, 78, 71, 0, 255], "image/png");
    image.setHeader("Content-ID", "<logo>");

    mime:Entity html = new;
    html.setText("<img src=\"cid:logo\">", "text/html");
    mime:Entity logo = new;
    logo.setByteArray([1, 2, 3], "image/png");
    logo.setHeader("Content-ID", "<logo>");
    mime:Entity related = new;
    related.setHeader("X-Part", "related");
    related.setBodyParts([html, logo], "multipart/related; type=\"text/html\"");

    mime:Entity noType = new;
    noType.setByteArray([65, 66]);
    noType.removeHeader("Content-Type");

    mime:Entity preEncoded = new;
    preEncoded.setText("pre");
    preEncoded.setHeader("Content-Transfer-Encoding", "base64");

    mime:Entity qpEncoded = new;
    qpEncoded.setText("qp=1");
    qpEncoded.setHeader("Content-Transfer-Encoding", "quoted-printable");

    mime:Entity binaryEncoded = new;
    binaryEncoded.setText("raw");
    binaryEncoded.setHeader("Content-Transfer-Encoding", "8bit");

    email:Message withAttachments = {
        to: ["bob@localhost", "Rob <rob@localhost>"],
        cc: "Carol <carol@localhost>",
        bcc: ["dave@localhost"],
        replyTo: "replies@localhost",
        sender: "sender@localhost",
        subject: "attachments",
        'from: "Alice <alice@localhost>",
        body: "See attached",
        attachments: [note, image, related, noType, preEncoded, qpEncoded, binaryEncoded,
            {filePath: "%[2]s", contentType: "application/json"}]
    };
    check smtp->sendMessage(withAttachments);
    io:println(withAttachments.attachments is (mime:Entity|email:Attachment)[]); // @output true

    email:Message single = {to: "bob@localhost", subject: "single-attachment", body: "Body", attachments: note};
    check smtp->sendMessage(single);
    // jBallerina rewrites the caller's single attachment into an array (ballerina-library#9744).
    io:println(single.attachments is mime:Entity); // @output true
    final readonly & email:Message frozen = {to: "bob@localhost", subject: "readonly", body: "Body",
        attachments: {filePath: "%[2]s", contentType: "application/json"}};
    check smtp->sendMessage(frozen);

    mime:Entity csv = new;
    csv.setText("a,b\n", "text/csv");
    check smtp->send("bob@localhost", "trailing-newline", "alice@localhost", "line1\nline2\n",
        htmlBody = "<p>x</p>\n", attachments = csv);

    mime:Entity resume = new;
    resume.setByteArray([37, 80, 68, 70], "application/pdf");
    resume.setHeader("Content-Disposition", "attachment; filename=\"résumé.pdf\"");
    check smtp->send("bob@localhost", "structured-headers", "alice@localhost", "Body",
        headers = {"Reply-To": "Jürgen <juergen@localhost>", "Bcc": "audit@localhost", "To": "Team <team@localhost>"},
        attachments = resume);

    check smtp->send("\"john doe\"@localhost", "quoted-local-part", "alice@localhost", "Body");

    // An empty custom address header clears that header and its recipients.
    check smtp->send("bob@localhost", "cleared-cc", "alice@localhost", "Body", cc = "carol@localhost",
        headers = {"Cc": ""});

    string[] many = [];
    foreach int i in 0 ..< 60 {
        many.push("user" + i.toString() + "@localhost");
    }
    check smtp->send(many, "%[4]s", "alice@localhost", "%[5]s");

    check smtp->send("bob@localhost", "send-attachment", "alice@localhost", "Body",
        attachments = {filePath: "%[2]s", contentType: "text/plain"});

    check smtp->sendMessage({to: "bob@localhost", subject: "html-only-attachment", htmlBody: "<p>café</p>",
        contentType: "text/html; charset=ISO-8859-1", attachments: note});
    check smtp->sendMessage({to: "bob@localhost", subject: "empty-typed", contentType: "text/html; charset=ISO-8859-1"});
    check smtp->send("bob@localhost", "trailing-semicolon", "alice@localhost", "Body", contentType = "text/plain;");
    check smtp->sendMessage({to: "bob@localhost", subject: "empty-from-header", body: "Body", headers: {"From": ""}});
    check smtp->send("bob@localhost", "quoted-at", "\"a@b\"@localhost", "Body");

    mime:Entity cafe = new;
    cafe.setText("café");
    mime:Entity binaryImage = new;
    binaryImage.setByteArray([137, 80, 78, 10, 0, 13, 65, 10, 66], "image/png");
    binaryImage.setHeader("Content-Transfer-Encoding", "binary");
    mime:Entity encodedRelated = new;
    encodedRelated.setBodyParts([html], "multipart/related");
    encodedRelated.setHeader("X-Part", "encoded");
    encodedRelated.setHeader("Content-Transfer-Encoding", "base64");
    mime:Entity named = new;
    named.setByteArray([37, 80, 68, 70], "application/pdf; name=\"résumé.pdf\"");
    mime:Entity popped = new;
    popped.setText("x");
    string[] types = check popped.getHeaders("Content-Type");
    _ = types.remove(0);
    check smtp->send("bob@localhost", "entity-encodings", "alice@localhost", "Body",
        attachments = [cafe, binaryImage, encodedRelated, named, popped]);
    io:println("sent"); // @output sent
}
`, server.port(), jsonFile, longText, longSubject, exactLines))

	got := messagesBySubject(t, server.received())
	expect := map[string]string{
		"text":          `text/plain; charset=us-ascii(7bit)"Hello\nWorld"`,
		"bare-cr":       `text/plain; charset=us-ascii(7bit)"a\nb\nc"`,
		"html":          `text/html; charset=us-ascii(7bit)"<b>Hi</b>"`,
		"html-as-plain": `text/plain; charset=us-ascii(7bit)"<b>Hi</b>"`,
		// #9738: the contentType charset applies to the text part of an alternative.
		"alternative":       `multipart/alternative[text/plain; charset=ISO-8859-1(quoted-printable)"caf\xe9" | text/html; charset=UTF-8(quoted-printable)"<p>café</p>"]`,
		"empty":             `text/plain; charset=us-ascii(7bit)""`,
		"unicode ü subject": `text/plain; charset=UTF-8(quoted-printable)"Grüße aus Köln"`,
		"日本語の件名":            fmt.Sprintf(`text/plain; charset=us-ascii(quoted-printable)%q`, longText),
		// #9739: a custom header replaces the generated one.
		"headers-overridden": `text/plain; charset=us-ascii(7bit)"Body"`,
		// Line breaks in header values are unfolded, so they cannot inject headers.
		"injected Bcc: eve@localhost": `text/plain; charset=us-ascii(7bit)"Body"`,
		// #9751: a body without htmlBody is a single part, not an alternative with an empty HTML part.
		"attachments": `multipart/mixed[text/plain; charset=us-ascii(7bit)"See attached" | ` +
			// #9737: every value of a multi-valued entity header is sent.
			`text/plain; charset=us-ascii(7bit){X-Tag=first,second}"note" | ` +
			`image/png(base64){Content-Id=<logo>}"\x89PNG\x00\xff" | ` +
			// #9736: a multipart entity keeps its subtype, parameters and headers.
			`multipart/related; type="text/html"{X-Part=related}[text/html; charset=us-ascii(7bit)"<img src=\"cid:logo\">" | image/png(base64){Content-Id=<logo>}"\x01\x02\x03"] | ` +
			`application/octet-stream(7bit)"AB" | text/plain; charset=us-ascii(base64)"pre" | text/plain; charset=us-ascii(quoted-printable)"qp=1" | text/plain; charset=us-ascii(8bit)"raw" | ` +
			// #9735: a file attachment uses the record's contentType.
			`application/json; name=data.json(7bit){Content-Disposition=attachment; filename=data.json}"{\"k\": 1}"]`,
		"single-attachment": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | text/plain; charset=us-ascii(7bit){X-Tag=first,second}"note"]`,
		"readonly":          `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | application/json; name=data.json(7bit){Content-Disposition=attachment; filename=data.json}"{\"k\": 1}"]`,
		// A part's final line break survives the multipart delimiter.
		"trailing-newline":             `multipart/mixed[multipart/alternative[text/plain; charset=us-ascii(7bit)"line1\nline2\n" | text/html; charset=us-ascii(7bit)"<p>x</p>\n"] | text/csv; charset=us-ascii(7bit)"a,b\n"]`,
		"structured-headers":           `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | application/pdf(7bit){Content-Disposition=attachment; filename*=utf-8''r%C3%A9sum%C3%A9.pdf}"%PDF"]`,
		"quoted-local-part":            `text/plain; charset=us-ascii(7bit)"Body"`,
		"cleared-cc":                   `text/plain; charset=us-ascii(7bit)"Body"`,
		strings.TrimSpace(longSubject): fmt.Sprintf(`text/plain; charset=us-ascii(7bit)%q`, strings.ReplaceAll(exactLines, `\r\n`, "\n")),
		"send-attachment":              `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | text/plain; name=data.json(7bit){Content-Disposition=attachment; filename=data.json}"{\"k\": 1}"]`,
		// With htmlBody alone, contentType applies to the HTML part, with or without attachments.
		"html-only-attachment": `multipart/mixed[text/html; charset=ISO-8859-1(quoted-printable)"<p>caf\xe9</p>" | text/plain; charset=us-ascii(7bit){X-Tag=first,second}"note"]`,
		// #9753: contentType applies to a message without a body.
		"empty-typed":        `text/html; charset=ISO-8859-1(7bit)""`,
		"trailing-semicolon": `text/plain; charset=us-ascii(7bit)"Body"`,
		"empty-from-header":  `text/plain; charset=us-ascii(7bit)"Body"`,
		"quoted-at":          `text/plain; charset=us-ascii(7bit)"Body"`,
		// #9754, #9755: text entities get a charset and binary data keeps its line-break bytes; a
		// multipart entity drops its transfer encoding, and the popped Content-Type falls back to octet-stream.
		"entity-encodings": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | text/plain; charset=UTF-8(quoted-printable)"café" | ` +
			`image/png(base64)"\x89PN\n\x00\rA\nB" | multipart/related{X-Part=encoded}[text/html; charset=us-ascii(7bit)"<img src=\"cid:logo\">"] | ` +
			`application/pdf; name*=utf-8''r%C3%A9sum%C3%A9.pdf(7bit)"%PDF" | application/octet-stream(7bit)"x"]`,
	}
	if len(got) != len(expect) {
		t.Errorf("received %d messages, want %d", len(got), len(expect))
	}
	for subject, want := range expect {
		r, ok := got[subject]
		if !ok {
			t.Errorf("no message with subject %q", subject)
			continue
		}
		if structure := mimeStructure(t, r.data); structure != want {
			t.Errorf("%s: structure\n got: %s\nwant: %s", subject, structure, want)
		}
		if r.authUser != "alice" {
			t.Errorf("%s: authenticated as %q, want alice", subject, r.authUser)
		}
	}

	headers := need(t, got, "headers-overridden").data
	if v := headerValues(t, headers, "From"); !slices.Equal(v, []string{"other@localhost"}) {
		t.Errorf("headers: From = %q, want one overridden value", v)
	}
	if v := headerValues(t, headers, "X-Trace"); !slices.Equal(v, []string{"42"}) {
		t.Errorf("headers: X-Trace = %q", v)
	}
	injected := need(t, got, "injected Bcc: eve@localhost").data
	if v := headerValues(t, injected, "X-Note"); !slices.Equal(v, []string{"line1 X-Injected: yes"}) {
		t.Errorf("injected: X-Note = %q", v)
	}
	for _, name := range []string{"Bcc", "X-Injected"} {
		if v := headerValues(t, injected, name); len(v) != 0 {
			t.Errorf("injected: %s header must not be present, got %q", name, v)
		}
	}
	if v := headerValues(t, need(t, got, "text").data, "From"); !slices.Equal(v, []string{"alice@localhost"}) {
		t.Errorf("text: From = %q", v)
	}
	if v := headerValues(t, need(t, got, "empty").data, "From"); !slices.Equal(v, []string{"alice"}) {
		t.Errorf("empty: From = %q, want the username as the default sender", v)
	}
	if v := headerValues(t, need(t, got, "empty-from-header").data, "From"); !slices.Equal(v, []string{"alice"}) {
		t.Errorf("empty-from-header: From = %q, want the username as the default sender", v)
	}
	if v := headerValues(t, need(t, got, "quoted-at").data, "Message-ID"); len(v) != 1 || strings.Count(v[0], "@") != 1 || !strings.HasSuffix(v[0], "@localhost>") {
		t.Errorf("quoted-at: Message-ID = %q, want the From domain", v)
	}
	if data := need(t, got, "entity-encodings").data; !strings.Contains(data, "Content-Type: application/pdf; name*=utf-8''r%C3%A9sum%C3%A9.pdf") {
		t.Errorf("entity-encodings: the non-ASCII Content-Type parameter is not RFC 2231 encoded:\n%s", data)
	}

	structured := need(t, got, "structured-headers")
	if v := headerValues(t, structured.data, "Reply-To"); len(v) != 1 {
		t.Errorf("structured: Reply-To = %q, want one value", v)
	} else if replyTo, err := mail.ParseAddress(v[0]); err != nil || replyTo.Name != "Jürgen" {
		t.Errorf("structured: Reply-To %q does not parse back (%v)", v, err)
	}
	if v := headerValues(t, structured.data, "Bcc"); len(v) != 0 {
		t.Errorf("structured: Bcc header must not be sent, got %q", v)
	}
	if v := headerValues(t, structured.data, "To"); !slices.Equal(v, []string{`"Team" <team@localhost>`}) {
		t.Errorf("structured: To = %q", v)
	}
	if want := []string{"team@localhost", "audit@localhost"}; !slices.Equal(structured.recipients, want) {
		t.Errorf("structured: recipients %q, want %q", structured.recipients, want)
	}
	quoted := need(t, got, "quoted-local-part")
	if want := []string{`"john doe"@localhost`}; !slices.Equal(quoted.recipients, want) {
		t.Errorf("quoted: recipients %q, want %q", quoted.recipients, want)
	}
	if v := headerValues(t, quoted.data, "To"); !slices.Equal(v, []string{`"john doe"@localhost`}) {
		t.Errorf("quoted: To = %q", v)
	}
	cleared := need(t, got, "cleared-cc")
	if v := headerValues(t, cleared.data, "Cc"); len(v) != 0 || !slices.Equal(cleared.recipients, []string{"bob@localhost"}) {
		t.Errorf("cleared-cc: Cc = %q, recipients %q", v, cleared.recipients)
	}
	folded := need(t, got, strings.TrimSpace(longSubject))
	for line := range strings.SplitSeq(folded.data, "\n") {
		if len(strings.TrimSuffix(line, "\r")) > 998 {
			t.Errorf("long headers: line of %d octets", len(line))
		}
	}
	if len(folded.recipients) != 60 {
		t.Errorf("long headers: %d recipients, want 60", len(folded.recipients))
	}

	attachments := need(t, got, "attachments")
	for name, want := range map[string]string{
		"From":     `"Alice" <alice@localhost>`,
		"To":       `bob@localhost, "Rob" <rob@localhost>`,
		"Cc":       `"Carol" <carol@localhost>`,
		"Reply-To": "replies@localhost",
		"Sender":   "sender@localhost",
	} {
		if v := headerValues(t, attachments.data, name); !slices.Equal(v, []string{want}) {
			t.Errorf("attachments: %s = %q, want %q", name, v, want)
		}
	}
	if v := headerValues(t, attachments.data, "Bcc"); len(v) != 0 {
		t.Errorf("attachments: Bcc header must not be sent, got %q", v)
	}
	wantRcpts := []string{"bob@localhost", "rob@localhost", "carol@localhost", "dave@localhost"}
	if !slices.Equal(attachments.recipients, wantRcpts) || attachments.from != "alice@localhost" {
		t.Errorf("attachments: envelope %q -> %q, want alice@localhost -> %q", attachments.from, attachments.recipients, wantRcpts)
	}
}

func TestEmailSmtpTLS(t *testing.T) {
	t.Parallel()
	caCertPEM, serverCertPEM, serverKeyPEM, _, _ := generateTestCerts(t)
	otherCACertPEM, _, _, _, _ := generateTestCerts(t)
	serverCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}
	serverTLS := &tls.Config{Certificates: []tls.Certificate{serverCert}}
	implicit := newFakeSmtpServer(t, smtpImplicitTLS, serverTLS, nil)
	startTLS := newFakeSmtpServer(t, smtpStartTLS, serverTLS, nil)
	plain := newFakeSmtpServer(t, smtpPlain, nil, nil)

	caBlock, _ := pem.Decode(caCertPEM)
	caFile := writeTempFile(t, "ca.pem", caCertPEM)
	derFile := writeTempFile(t, "ca.der", caBlock.Bytes)
	// #9734: the trusted CA is the second certificate in the bundle.
	bundleFile := writeTempFile(t, "bundle.pem", append(append([]byte{}, otherCACertPEM...), caCertPEM...))
	otherCAFile := writeTempFile(t, "other-ca.pem", otherCACertPEM)
	garbageFile := writeTempFile(t, "garbage.pem", []byte("not a certificate"))

	runEmailCase(t, "email-smtp-tls-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;

function trySend(string label, string host, int port, email:Security security, email:SecureSocket? secureSocket) {
    email:SmtpClient|email:Error smtp = secureSocket is email:SecureSocket
        ? new (host, "alice", "secret", port = port, security = security, secureSocket = secureSocket)
        : new (host, "alice", "secret", port = port, security = security);
    if smtp is email:Error {
        string message = smtp.message();
        // The parse error names the certificate's temporary path, so only its prefix is stable.
        io:println(label, ": init error: ", message.indexOf("Could not parse certificate: ") == 0
            ? "Could not parse certificate" : message);
        return;
    }
    email:Error? result = smtp->send("bob@localhost", label, "alice@localhost", "Body");
    io:println(label, ": ", result is email:Error ? withoutPort(result.message()) : "sent");
}

function withoutPort(string message) returns string {
    int? index = message.indexOf(", port: ");
    return index is int ? message.substring(0, index) : message;
}

public function main() {
    // #9731, #9733: no protocol and no versions use the platform defaults.
    trySend("ssl-default-protocol", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s"}); // @output ssl-default-protocol: sent
    trySend("ssl-der-cert", "127.0.0.1", %[1]s, email:SSL, {cert: "%[5]s"}); // @output ssl-der-cert: sent
    trySend("ssl-bundle", "127.0.0.1", %[1]s, email:SSL, {cert: "%[6]s"}); // @output ssl-bundle: sent
    trySend("ssl-tls13", "127.0.0.1", %[1]s, email:SSL,
        {cert: "%[4]s", protocol: {name: email:TLS, versions: ["TLSv1.2", "TLSv1.3"]}}); // @output ssl-tls13: sent
    trySend("ssl-cipher", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s",
        protocol: {name: email:TLS, versions: ["TLSv1.2"]}, ciphers: ["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"]}); // @output ssl-cipher: sent
    // A listed suite that Go's TLS library lacks is skipped while another one is usable.
    trySend("ssl-cipher-mixed", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", protocol: {name: email:TLS, versions: ["TLSv1.2"]},
        ciphers: ["TLS_DHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"]}); // @output ssl-cipher-mixed: sent
    // Without a TLS 1.3 suite in ciphers, TLS 1.3 is not negotiated.
    trySend("ssl-cipher-tls12", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", ciphers: ["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"]}); // @output ssl-cipher-tls12: sent
    trySend("ssl-cipher-tls13", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s",
        ciphers: ["TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384", "TLS_CHACHA20_POLY1305_SHA256"]}); // @output ssl-cipher-tls13: sent
    trySend("ssl-cipher-tls13-subset", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", protocol: {name: email:TLS, versions: ["TLSv1.3"]},
        ciphers: ["TLS_AES_128_GCM_SHA256"]}); // @output ssl-cipher-tls13-subset: init error: TLS 1.3 cipher suites cannot be restricted; list all of TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256 or none
    trySend("ssl-untrusted", "127.0.0.1", %[1]s, email:SSL, {cert: "%[7]s"}); // @output ssl-untrusted: Could not connect to SMTP host: 127.0.0.1
    trySend("ssl-system-trust", "127.0.0.1", %[1]s, email:SSL, ()); // @output ssl-system-trust: Could not connect to SMTP host: 127.0.0.1
    trySend("ssl-hostname", "localhost", %[1]s, email:SSL, {cert: "%[4]s"}); // @output ssl-hostname: Could not connect to SMTP host: localhost
    trySend("ssl-no-hostname-check", "localhost", %[1]s, email:SSL, {cert: "%[4]s", verifyHostName: false}); // @output ssl-no-hostname-check: sent

    // #9729: START_TLS_ALWAYS upgrades a plaintext connection with STARTTLS.
    trySend("starttls-always", "127.0.0.1", %[2]s, email:START_TLS_ALWAYS, {cert: "%[4]s"}); // @output starttls-always: sent
    trySend("starttls-auto", "127.0.0.1", %[2]s, email:START_TLS_AUTO, {cert: "%[4]s"}); // @output starttls-auto: sent
    trySend("starttls-untrusted", "127.0.0.1", %[2]s, email:START_TLS_ALWAYS, {cert: "%[7]s"}); // @output starttls-untrusted: Could not convert socket to TLS
    trySend("starttls-always-unsupported", "127.0.0.1", %[3]s, email:START_TLS_ALWAYS, {cert: "%[4]s"}); // @output starttls-always-unsupported: STARTTLS is required but host does not support STARTTLS
    trySend("starttls-auto-unsupported", "127.0.0.1", %[3]s, email:START_TLS_AUTO, ()); // @output starttls-auto-unsupported: sent
    // #9730: secureSocket does not override the security mode.
    trySend("never-with-secure-socket", "127.0.0.1", %[3]s, email:START_TLS_NEVER, {cert: "%[4]s"}); // @output never-with-secure-socket: sent
    trySend("never", "127.0.0.1", %[2]s, email:START_TLS_NEVER, ()); // @output never: sent

    // #9732: configuration errors are returned, not panics.
    trySend("missing-cert", "127.0.0.1", %[1]s, email:SSL, {cert: "/no/such/cert.pem"}); // @output missing-cert: init error: /no/such/cert.pem (No such file or directory)
    trySend("garbage-cert", "127.0.0.1", %[1]s, email:SSL, {cert: "%[8]s"}); // @output garbage-cert: init error: Could not parse certificate
    trySend("bad-version", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", protocol: {name: email:TLS, versions: ["SSLv3"]}}); // @output bad-version: init error: Unsupported TLS protocol version: SSLv3
    trySend("bad-cipher", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", ciphers: ["NOT_A_CIPHER"]}); // @output bad-cipher: init error: Unsupported cipher suite: NOT_A_CIPHER
    // Only suites that Java knows are skipped, so a misspelt name beside a usable suite is still an error.
    trySend("bad-cipher-mixed", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", protocol: {name: email:TLS, versions: ["TLSv1.2"]},
        ciphers: ["NOT_A_CIPHER", "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"]}); // @output bad-cipher-mixed: init error: Unsupported cipher suite: NOT_A_CIPHER
    trySend("ssl-cipher-tls13-repeated", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", protocol: {name: email:TLS, versions: ["TLSv1.3"]},
        ciphers: ["TLS_AES_128_GCM_SHA256", "TLS_AES_128_GCM_SHA256", "TLS_AES_128_GCM_SHA256"]}); // @output ssl-cipher-tls13-repeated: init error: TLS 1.3 cipher suites cannot be restricted; list all of TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256 or none
    trySend("ssl-tls13-only-tls12-cipher", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", protocol: {name: email:TLS, versions: ["TLSv1.3"]},
        ciphers: ["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"]}); // @output ssl-tls13-only-tls12-cipher: init error: The cipher suites cannot be used with the TLS protocol versions
    trySend("ssl-tls12-only-tls13-ciphers", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", protocol: {name: email:TLS, versions: ["TLSv1.2"]},
        ciphers: ["TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384", "TLS_CHACHA20_POLY1305_SHA256"]}); // @output ssl-tls12-only-tls13-ciphers: init error: The cipher suites cannot be used with the TLS protocol versions
    // GCM suites need TLS 1.2, so TLS 1.1 alone leaves no usable suite.
    trySend("ssl-tls11-only-gcm-cipher", "127.0.0.1", %[1]s, email:SSL, {cert: "%[4]s", protocol: {name: email:TLS, versions: ["TLSv1.1"]},
        ciphers: ["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"]}); // @output ssl-tls11-only-gcm-cipher: init error: The cipher suites cannot be used with the TLS protocol versions
}
`, implicit.port(), startTLS.port(), plain.port(), caFile, derFile, bundleFile, otherCAFile, garbageFile))

	implicitGot := messagesBySubject(t, implicit.received())
	startTLSGot := messagesBySubject(t, startTLS.received())
	plainGot := messagesBySubject(t, plain.received())
	for subject, want := range map[string]uint16{"ssl-cipher-tls12": tls.VersionTLS12, "ssl-cipher-tls13": tls.VersionTLS13} {
		if got := need(t, implicitGot, subject).tlsVersion; got != want {
			t.Errorf("%s: negotiated %s, want %s", subject, tls.VersionName(got), tls.VersionName(want))
		}
	}
	// Each subject is looked up on the relay it was sent to, so "never" shows the client stayed
	// plaintext although that relay offered STARTTLS.
	for _, c := range []struct {
		subject string
		relay   map[string]*smtpReceived
		wantTLS bool
	}{
		{"ssl-default-protocol", implicitGot, true},
		{"starttls-always", startTLSGot, true},
		{"starttls-auto", startTLSGot, true},
		{"never", startTLSGot, false},
		{"starttls-auto-unsupported", plainGot, false},
		{"never-with-secure-socket", plainGot, false},
	} {
		if got := need(t, c.relay, c.subject).tls; got != c.wantTLS {
			t.Errorf("%s: tls=%v, want %v", c.subject, got, c.wantTLS)
		}
	}
}

func TestEmailSmtpServerErrors(t *testing.T) {
	t.Parallel()
	rejecting := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) {
		s.rejectRcpt = "nobody@localhost"
		s.rejectFrom = "blocked@localhost"
	})
	rejectData := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.rejectData = true })
	plainAuth := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.authMechanisms = "PLAIN" })
	cramOnly := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.authMechanisms = "CRAM-MD5" })
	noAuth := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) {
		s.authMechanisms = ""
		s.rejectEhlo = true
	})
	legacyAuth := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.legacyAuth = true })
	dropAfterMail := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.dropAfterMail = true })
	utf8Server := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.smtpUTF8 = true })
	bareReply := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.bareDataReply = true })
	closing := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.closeRcpt = "bob@localhost" })
	closingEhlo := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.closeEhlo = true })
	// A reply line over 4096 bytes, and a reply over 64 KiB, each followed by a valid greeting line.
	longLine := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) {
		s.greeting = "220 " + strings.Repeat("x", 4096) + "\r\n"
	})
	longReply := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) {
		s.greeting = strings.Repeat("220-"+strings.Repeat("x", 1000)+"\r\n", 70) + "220 ready\r\n"
	})
	// The PAL refuses this address, so no other listener can take the port during the test.
	refusing := newEmailPal()
	refusing.refused = "127.0.0.1:9"

	runEmailCaseWith(t, refusing, "email-smtp-server-errors-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/mime;

function smtpClient(int port, string password = "secret") returns email:SmtpClient|error =>
    new ("127.0.0.1", "alice", password, port = port, security = email:START_TLS_NEVER);

function report(string label, email:Error? result) {
    string message = result is email:Error ? result.message() : "sent";
    int? index = message.indexOf(", port: ");
    io:println(label, ": [", index is int ? message.substring(0, index) : message, "]");
}

public function main() returns error? {
    email:SmtpClient rejecting = check smtpClient(%[1]s);
    email:Error? rejectedRcpt = rejecting->send(["bob@localhost", "nobody@localhost"], "s", "alice@localhost", "b");
    report("rejected-rcpt", rejectedRcpt); // @output rejected-rcpt: [Error while sending the message to SMTP server : Invalid Addresses nobody@localhost]
    email:Error? rejectedFrom = rejecting->send("bob@localhost", "s", "blocked@localhost", "b");
    report("rejected-from", rejectedFrom); // @output rejected-from: [Error while sending the message to SMTP server : 550 sender rejected ]
    email:Error? acceptedAfterReject = rejecting->send("bob@localhost", "s", "alice@localhost", "b");
    report("accepted-after-reject", acceptedAfterReject); // @output accepted-after-reject: [sent]

    email:SmtpClient rejectData = check smtpClient(%[2]s);
    email:Error? rejectedData = rejectData->send("bob@localhost", "s", "alice@localhost", "b");
    report("rejected-data", rejectedData); // @output rejected-data: [Error while sending the message to SMTP server : 554 message rejected ]

    email:SmtpClient plainAuth = check smtpClient(%[3]s);
    email:Error? authPlain = plainAuth->send("bob@localhost", "s", "alice@localhost", "b");
    report("auth-plain", authPlain); // @output auth-plain: [sent]
    email:SmtpClient wrongPassword = check smtpClient(%[3]s, "wrong");
    email:Error? authFailed = wrongPassword->send("bob@localhost", "s", "alice@localhost", "b");
    report("auth-failed", authFailed); // @output auth-failed: [535 5.7.8 Authentication failed]

    email:SmtpClient cramOnly = check smtpClient(%[4]s);
    email:Error? authUnsupported = cramOnly->send("bob@localhost", "s", "alice@localhost", "b");
    report("auth-unsupported", authUnsupported); // @output auth-unsupported: [No authentication mechanisms supported by both server and client]

    email:SmtpClient noAuthServer = check smtpClient(%[5]s);
    email:Error? heloWithoutAuth = noAuthServer->send("bob@localhost", "s", "alice@localhost", "b");
    report("helo-without-auth", heloWithoutAuth); // @output helo-without-auth: [sent]
    email:SmtpClient anonymous = check new ("127.0.0.1", port = %[5]s, security = email:START_TLS_NEVER);
    email:Error? anonymousResult = anonymous->send("bob@localhost", "s", "alice@localhost", "b");
    report("anonymous", anonymousResult); // @output anonymous: [sent]

    email:SmtpClient legacyAuth = check smtpClient(%[6]s);
    email:Error? legacy = legacyAuth->send("bob@localhost", "s", "alice@localhost", "b");
    report("legacy-auth-login", legacy); // @output legacy-auth-login: [sent]

    // A dropped connection is an I/O failure, not a rejected recipient.
    email:SmtpClient dropping = check smtpClient(%[7]s);
    email:Error? dropped = dropping->send("bob@localhost", "s", "alice@localhost", "b");
    report("dropped-after-mail", dropped); // @output dropped-after-mail: [Exception reading response]

    // Non-ASCII addresses need the SMTPUTF8 extension; jBallerina sends them without it (ballerina-library#9756).
    email:Error? utf8Unsupported = plainAuth->send("jösé@localhost", "s", "alice@localhost", "b");
    report("utf8-unsupported", utf8Unsupported); // @output utf8-unsupported: [SMTPUTF8 is required for non-ASCII addresses but host does not support SMTPUTF8]
    email:SmtpClient utf8Server = check smtpClient(%[8]s);
    email:Error? utf8 = utf8Server->send("jösé@localhost", "s", "alice@localhost", "b");
    report("utf8", utf8); // @output utf8: [sent]
    // jBallerina writes a non-ASCII message ID without SMTPUTF8 (ballerina-library#9905).
    email:Error? idUnsupported = plainAuth->send("bob@localhost", "s", "alice@localhost", "b", headers = {"In-Reply-To": "<réponse@example.com>"});
    report("message-id-utf8-unsupported", idUnsupported); // @output message-id-utf8-unsupported: [SMTPUTF8 is required for non-ASCII message IDs but host does not support SMTPUTF8]
    email:Error? idUtf8 = utf8Server->send("bob@localhost", "message-id-utf8", "alice@localhost", "b", headers = {"In-Reply-To": "<réponse@example.com>"});
    report("message-id-utf8", idUtf8); // @output message-id-utf8: [sent]
    mime:Entity replyPart = new;
    replyPart.setText("x");
    replyPart.setHeader("References", "<réponse@example.com>");
    replyPart.setHeader("Content-ID", "<imagé@example.com>");
    email:Error? nestedIdUnsupported = plainAuth->send("bob@localhost", "s", "alice@localhost", "b", attachments = replyPart);
    report("nested-message-id-utf8-unsupported", nestedIdUnsupported); // @output nested-message-id-utf8-unsupported: [SMTPUTF8 is required for non-ASCII message IDs but host does not support SMTPUTF8]
    email:Error? nestedIdUtf8 = utf8Server->send("bob@localhost", "nested-message-id-utf8", "alice@localhost", "b", attachments = replyPart);
    report("nested-message-id-utf8", nestedIdUtf8); // @output nested-message-id-utf8: [sent]

    // RFC 5321 allows a reply that is only its code.
    email:SmtpClient bareReply = check smtpClient(%[9]s);
    email:Error? bare = bareReply->send("bob@localhost", "s", "alice@localhost", "b");
    report("bare-reply-code", bare); // @output bare-reply-code: [sent]
    // jBallerina tries the next command after 421 and reports the I/O failure (ballerina-library#9897).
    email:SmtpClient closing = check smtpClient(%[10]s);
    email:Error? closedSession = closing->send(["bob@localhost", "carol@localhost"], "s", "alice@localhost", "b");
    report("rcpt-421", closedSession); // @output rcpt-421: [Error while sending the message to SMTP server : 421 service closing ]
    email:SmtpClient closingEhlo = check smtpClient(%[11]s);
    email:Error? ehloClosed = closingEhlo->send("bob@localhost", "s", "alice@localhost", "b");
    report("ehlo-421", ehloClosed); // @output ehlo-421: [421 service closing]

    // An oversized reply line or reply fails the send instead of being buffered.
    email:SmtpClient longLine = check smtpClient(%[12]s);
    email:Error? lineTooLong = longLine->send("bob@localhost", "s", "alice@localhost", "b");
    report("long-reply-line", lineTooLong); // @output long-reply-line: [Could not connect to SMTP host: 127.0.0.1]
    io:println(lineTooLong is email:Error ? lineTooLong.cause() : ()); // @output error("response line too long")
    email:SmtpClient longReply = check smtpClient(%[13]s);
    email:Error? replyTooLong = longReply->send("bob@localhost", "s", "alice@localhost", "b");
    report("long-reply", replyTooLong); // @output long-reply: [Could not connect to SMTP host: 127.0.0.1]
    io:println(replyTooLong is email:Error ? replyTooLong.cause() : ()); // @output error("response too long")

    email:SmtpClient closed = check smtpClient(9);
    email:Error? refused = closed->send("bob@localhost", "s", "alice@localhost", "b");
    report("refused", refused); // @output refused: [Could not connect to SMTP host: 127.0.0.1]
    io:println(refused is email:Error && refused.cause() is error); // @output true
}
`, rejecting.port(), rejectData.port(), plainAuth.port(), cramOnly.port(), noAuth.port(), legacyAuth.port(), dropAfterMail.port(), utf8Server.port(),
		bareReply.port(), closing.port(), closingEhlo.port(), longLine.port(), longReply.port()))

	utf8Received := utf8Server.received()
	if len(utf8Received) != 3 || utf8Received[0].mailParams != "SMTPUTF8" || !slices.Equal(utf8Received[0].recipients, []string{"jösé@localhost"}) {
		t.Errorf("SMTPUTF8 server: received %d messages, want the first sent with SMTPUTF8 to jösé@localhost", len(utf8Received))
	}
	if r := need(t, messagesBySubject(t, utf8Received), "message-id-utf8"); r.mailParams != "SMTPUTF8" || !slices.Equal(headerValues(t, r.data, "In-Reply-To"), []string{"<réponse@example.com>"}) {
		t.Errorf("non-ASCII In-Reply-To: MAIL params %q, header %q; want SMTPUTF8 and the ID unchanged", r.mailParams, headerValues(t, r.data, "In-Reply-To"))
	}
	if r := need(t, messagesBySubject(t, utf8Received), "nested-message-id-utf8"); r.mailParams != "SMTPUTF8" || !strings.Contains(r.data, "References: <réponse@example.com>") || !strings.Contains(r.data, "Content-ID: <imagé@example.com>") {
		t.Errorf("non-ASCII attachment References and Content-ID: MAIL params %q, want SMTPUTF8 and the IDs unchanged:\n%s", r.mailParams, r.data)
	}

	if got := len(rejecting.received()); got != 1 {
		t.Errorf("rejecting server accepted %d messages, want 1", got)
	}
	for _, r := range plainAuth.received() {
		if r.authUser != "alice" {
			t.Errorf("PLAIN auth: authenticated as %q", r.authUser)
		}
	}
	for _, r := range legacyAuth.received() {
		if r.authUser != "alice" {
			t.Errorf("AUTH=LOGIN: authenticated as %q", r.authUser)
		}
	}
	if len(legacyAuth.received()) != 1 {
		t.Errorf("AUTH=LOGIN server received %d messages, want 1", len(legacyAuth.received()))
	}
	for _, r := range noAuth.received() {
		if r.authUser != "" {
			t.Errorf("no-AUTH server: authenticated as %q", r.authUser)
		}
	}
}

func TestEmailSmtpTransferEncodings(t *testing.T) {
	t.Parallel()
	eightBit := newFakeSmtpServer(t, smtpPlain, nil, func(s *fakeSmtpServer) { s.eightBitMIME = true })
	sevenBit := newFakeSmtpServer(t, smtpPlain, nil, nil)
	utf16File := writeTempFile(t, "utf16.txt", []byte{0x4E, 0x0A, 0x00, 0x78})
	// Sending to sevenBit encodes the message twice, and both encodings must use one read of the file.
	p := newEmailPal()
	p.readOnce = writeTempFile(t, "once.txt", []byte("once"))
	longWord := strings.Repeat("a", 1200)
	messageID := "<" + strings.Repeat("a", 1000) + "@localhost>"
	runEmailCaseWith(t, p, "email-smtp-transfer-encodings-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/mime;

function eightBitPart() returns mime:Entity {
    mime:Entity part = new;
    part.setText("café");
    part.setHeader("Content-Transfer-Encoding", "8bit");
    return part;
}

public function main() returns error? {
    email:SmtpClient eightBit = check new ("127.0.0.1", "alice", "secret", port = %[1]s, security = email:START_TLS_NEVER);
    email:SmtpClient sevenBit = check new ("127.0.0.1", "alice", "secret", port = %[2]s, security = email:START_TLS_NEVER);
    // jBallerina sends 8-bit data without BODY=8BITMIME, even to a server without 8BITMIME (ballerina-library#9807).
    check eightBit->send("bob@localhost", "8bitmime", "alice@localhost", "Body", attachments = eightBitPart());
    check sevenBit->send("bob@localhost", "no-8bitmime", "alice@localhost", "Body", attachments = eightBitPart());

    mime:Entity qpImage = new;
    qpImage.setByteArray([65, 10, 66, 13, 67], "image/png");
    qpImage.setHeader("Content-Transfer-Encoding", "quoted-printable");
    check sevenBit->send("bob@localhost", "qp-binary", "alice@localhost", "Body", attachments = qpImage);

    check sevenBit->send("bob@localhost", "utf-16", "alice@localhost", "上x", contentType = "text/plain; charset=UTF-16BE");
    check sevenBit->send("bob@localhost", "charset-param", "alice@localhost", "Body", contentType = "text/plain; charset=UTF-8; name=\"résumé.txt\"");

    check sevenBit->send("bob@localhost", "quoted-comma", "alice@localhost", "Body",
        headers = {"To": "\"Doe, Jane\" <jane@localhost>, localuser"});
    // An escaped parenthesis and a quote inside a comment neither end it nor start a quoted string.
    check sevenBit->send("bob@localhost", "comment-comma", "alice@localhost", "Body",
        headers = {"To": "\"Doe, Jane\" <jane@localhost> (x\\) \" y, z), localuser"});

    // UTF-16 attachments are binary data, so their 0x0A bytes are not rewritten as line endings.
    mime:Entity utf16Entity = new;
    utf16Entity.setByteArray([0x4E, 0x0A, 0x00, 0x78], "text/plain; charset=UTF-16BE");
    utf16Entity.setHeader("Content-Transfer-Encoding", "quoted-printable");
    check sevenBit->send("bob@localhost", "utf-16-attachments", "alice@localhost", "Body",
        attachments = [{filePath: "%[3]s", contentType: "text/plain; charset=UTF-16BE"}, utf16Entity]);

    mime:Entity binaryLabel = new;
    binaryLabel.setByteArray([65, 66], "application/octet-stream");
    binaryLabel.setHeader("Content-Transfer-Encoding", "binary");
    check eightBit->send("bob@localhost", "binary-label", "alice@localhost", "Body", attachments = binaryLabel);

    // A 7-bit build for a server without 8BITMIME relabels a binary part too.
    check sevenBit->send("bob@localhost", "binary-fallback", "alice@localhost", "Body", attachments = [binaryLabel, eightBitPart()]);

    mime:Entity spaceText = new;
    spaceText.setText("x");
    mime:Entity spaceBoundary = new;
    spaceBoundary.setBodyParts([spaceText], "multipart/mixed; boundary=\"demo \"");
    check sevenBit->send("bob@localhost", "invalid-boundary", "alice@localhost", "Body", attachments = spaceBoundary);

    // Encoded-words are not allowed in a message id, so a long one is sent as it is, as jBallerina does.
    check sevenBit->send("bob@localhost", "in-reply-to", "alice@localhost", "Body", headers = {"In-Reply-To": "%[6]s"});

    mime:Entity delimiterText = new;
    delimiterText.setText("before\n--demo--\nafter");
    mime:Entity demo = new;
    demo.setBodyParts([delimiterText], "multipart/mixed; boundary=demo");
    check sevenBit->send("bob@localhost", "boundary-in-part", "alice@localhost", "Body", attachments = demo);

    check sevenBit->send("bob@localhost", "%[5]s", "alice@localhost", "Body");

    check sevenBit->send("bob@localhost", "read-once", "alice@localhost", "Body",
        attachments = [eightBitPart(), {filePath: "%[4]s", contentType: "text/plain"}]);
    io:println("sent"); // @output sent
}
`, eightBit.port(), sevenBit.port(), utf16File, p.readOnce, longWord, messageID))

	got := messagesBySubject(t, append(eightBit.received(), sevenBit.received()...))
	expect := map[string]string{
		"8bitmime":    `multipart/mixed(8bit)[text/plain; charset=us-ascii(7bit)"Body" | text/plain; charset=UTF-8(8bit)"café"]`,
		"no-8bitmime": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | text/plain; charset=UTF-8(quoted-printable)"café"]`,
		"qp-binary":   `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | image/png(quoted-printable)"A\nB\rC"]`,
		// UTF-16 output is binary data, so its 0x0A byte is not rewritten as a line ending.
		"utf-16":        `text/plain; charset=UTF-16BE(base64)"N\n\x00x"`,
		"charset-param": `text/plain; charset=UTF-8; name*=utf-8''r%C3%A9sum%C3%A9.txt(7bit)"Body"`,
		"quoted-comma":  `text/plain; charset=us-ascii(7bit)"Body"`,
		"comment-comma": `text/plain; charset=us-ascii(7bit)"Body"`,
		"utf-16-attachments": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | ` +
			`text/plain; charset=UTF-16BE; name=utf16.txt(base64){Content-Disposition=attachment; filename=utf16.txt}"N\n\x00x" | ` +
			`text/plain; charset=UTF-16BE(quoted-printable)"N\n\x00x"]`,
		// #9890: DATA cannot carry the binary encoding, so a line-safe part is relabelled.
		"binary-label": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | application/octet-stream(7bit)"AB"]`,
		"binary-fallback": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | application/octet-stream(7bit)"AB" | ` +
			`text/plain; charset=UTF-8(quoted-printable)"café"]`,
		"invalid-boundary": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | multipart/mixed[text/plain; charset=us-ascii(7bit)"x"]]`,
		"boundary-in-part": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | multipart/mixed[text/plain; charset=us-ascii(7bit)"before\n--demo--\nafter"]]`,
		longWord:           `text/plain; charset=us-ascii(7bit)"Body"`,
		"read-once": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | text/plain; charset=UTF-8(quoted-printable)"café" | ` +
			`text/plain; name=once.txt(7bit){Content-Disposition=attachment; filename=once.txt}"once"]`,
	}
	for subject, want := range expect {
		r, ok := got[subject]
		if !ok {
			t.Errorf("no message with subject %q", subject)
			continue
		}
		if structure := mimeStructure(t, r.data); structure != want {
			t.Errorf("%s: structure\n got: %s\nwant: %s", subject, structure, want)
		}
	}
	if params := need(t, got, "8bitmime").mailParams; params != "BODY=8BITMIME" {
		t.Errorf("8bitmime: MAIL parameters %q, want BODY=8BITMIME", params)
	}
	if r := need(t, got, "no-8bitmime"); r.mailParams != "" || strings.ContainsFunc(r.data, func(c rune) bool { return c >= 0x80 }) {
		t.Errorf("no-8bitmime: MAIL parameters %q, want none and no 8-bit data:\n%s", r.mailParams, r.data)
	}
	// A non-ASCII parameter is sent in RFC 2231 form, so the header stays ASCII without SMTPUTF8.
	if v := headerValues(t, need(t, got, "charset-param").data, "Content-Type"); len(v) != 1 || strings.ContainsFunc(v[0], func(c rune) bool { return c >= 0x80 }) {
		t.Errorf("charset-param: Content-Type %q, want ASCII", v)
	}
	if data := need(t, got, "qp-binary").data; !strings.Contains(data, "A=0AB=0DC") {
		t.Errorf("qp-binary: line-break bytes are not encoded:\n%s", data)
	}
	for _, subject := range []string{"quoted-comma", "comment-comma"} {
		if rcpts := need(t, got, subject).recipients; !slices.Equal(rcpts, []string{"jane@localhost", "localuser"}) {
			t.Errorf("%s: recipients %q", subject, rcpts)
		}
	}
	if data := need(t, got, "utf-16-attachments").data; !strings.Contains(data, "N=0A=00x") {
		t.Errorf("utf-16-attachments: the quoted-printable entity's 0x0A byte is not encoded:\n%s", data)
	}
	if data := need(t, got, "invalid-boundary").data; strings.Contains(data, `boundary="demo "`) {
		t.Errorf("invalid-boundary: the boundary ending in a space is reused:\n%s", data)
	}
	msg, err := mail.ReadMessage(strings.NewReader(need(t, got, "in-reply-to").data))
	if err != nil {
		t.Fatalf("parsing in-reply-to: %v", err)
	}
	if id := msg.Header.Get("In-Reply-To"); id != messageID {
		t.Errorf("in-reply-to: header %q, want the id unchanged", id)
	}
	// #9891: a subject word longer than a header line is split into encoded-words that folding can wrap.
	for line := range strings.SplitSeq(need(t, got, longWord).data, "\n") {
		if len(strings.TrimSuffix(line, "\r")) > 998 {
			t.Errorf("long word: line of %d octets", len(line))
		}
	}
}
