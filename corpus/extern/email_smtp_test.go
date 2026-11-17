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

package extern_test

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
}

func newEmailPal() *emailPal {
	return &emailPal{TestPal: testharness.NewTestPal()}
}

func (p *emailPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.Net = pal.Net{Dial: palnative.Dial, UpgradeTLS: palnative.UpgradeTLS}
	return base
}

func runEmailCase(t *testing.T, name, balContent string) {
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
	runExtern(t, tc, newEmailPal(), nil)
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
		out[subject] = r
	}
	return out
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
	text := strings.ReplaceAll(string(decoded), "\r\n", "\n")
	if topLevel {
		text = strings.TrimSuffix(text, "\n")
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
		// A body without htmlBody is a single part, not an alternative with an empty HTML part.
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
		"empty-typed":          `text/html; charset=ISO-8859-1(7bit)""`,
		"trailing-semicolon":   `text/plain; charset=us-ascii(7bit)"Body"`,
		"empty-from-header":    `text/plain; charset=us-ascii(7bit)"Body"`,
		"quoted-at":            `text/plain; charset=us-ascii(7bit)"Body"`,
		// Text entities get a charset, binary data keeps its line-break bytes, a multipart entity
		// drops its transfer encoding, and the popped Content-Type falls back to octet-stream.
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

	headers := got["headers-overridden"].data
	if v := headerValues(t, headers, "From"); !slices.Equal(v, []string{"other@localhost"}) {
		t.Errorf("headers: From = %q, want one overridden value", v)
	}
	if v := headerValues(t, headers, "X-Trace"); !slices.Equal(v, []string{"42"}) {
		t.Errorf("headers: X-Trace = %q", v)
	}
	injected := got["injected Bcc: eve@localhost"].data
	if v := headerValues(t, injected, "X-Note"); !slices.Equal(v, []string{"line1 X-Injected: yes"}) {
		t.Errorf("injected: X-Note = %q", v)
	}
	for _, name := range []string{"Bcc", "X-Injected"} {
		if v := headerValues(t, injected, name); len(v) != 0 {
			t.Errorf("injected: %s header must not be present, got %q", name, v)
		}
	}
	if v := headerValues(t, got["text"].data, "From"); !slices.Equal(v, []string{"alice@localhost"}) {
		t.Errorf("text: From = %q", v)
	}
	if v := headerValues(t, got["empty"].data, "From"); !slices.Equal(v, []string{"alice"}) {
		t.Errorf("empty: From = %q, want the username as the default sender", v)
	}
	if v := headerValues(t, got["empty-from-header"].data, "From"); !slices.Equal(v, []string{"alice"}) {
		t.Errorf("empty-from-header: From = %q, want the username as the default sender", v)
	}
	if v := headerValues(t, got["quoted-at"].data, "Message-ID"); len(v) != 1 || strings.Count(v[0], "@") != 1 || !strings.HasSuffix(v[0], "@localhost>") {
		t.Errorf("quoted-at: Message-ID = %q, want the From domain", v)
	}
	if data := got["entity-encodings"].data; !strings.Contains(data, "Content-Type: application/pdf; name*=utf-8''r%C3%A9sum%C3%A9.pdf") {
		t.Errorf("entity-encodings: the non-ASCII Content-Type parameter is not RFC 2231 encoded:\n%s", data)
	}

	structured := got["structured-headers"]
	if replyTo, err := mail.ParseAddress(headerValues(t, structured.data, "Reply-To")[0]); err != nil || replyTo.Name != "Jürgen" {
		t.Errorf("structured: Reply-To %q does not parse back (%v)", headerValues(t, structured.data, "Reply-To"), err)
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
	quoted := got["quoted-local-part"]
	if want := []string{`"john doe"@localhost`}; !slices.Equal(quoted.recipients, want) {
		t.Errorf("quoted: recipients %q, want %q", quoted.recipients, want)
	}
	if v := headerValues(t, quoted.data, "To"); !slices.Equal(v, []string{`"john doe"@localhost`}) {
		t.Errorf("quoted: To = %q", v)
	}
	cleared := got["cleared-cc"]
	if v := headerValues(t, cleared.data, "Cc"); len(v) != 0 || !slices.Equal(cleared.recipients, []string{"bob@localhost"}) {
		t.Errorf("cleared-cc: Cc = %q, recipients %q", v, cleared.recipients)
	}
	folded := got[strings.TrimSpace(longSubject)]
	for line := range strings.SplitSeq(folded.data, "\n") {
		if len(strings.TrimSuffix(line, "\r")) > 998 {
			t.Errorf("long headers: line of %d octets", len(line))
		}
	}
	if len(folded.recipients) != 60 {
		t.Errorf("long headers: %d recipients, want 60", len(folded.recipients))
	}

	attachments := got["attachments"]
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
}
`, implicit.port(), startTLS.port(), plain.port(), caFile, derFile, bundleFile, otherCAFile, garbageFile))

	tlsBySubject := map[string]bool{}
	for _, server := range []*fakeSmtpServer{implicit, startTLS, plain} {
		for subject, r := range messagesBySubject(t, server.received()) {
			tlsBySubject[subject] = r.tls
		}
	}
	for subject, wantTLS := range map[string]bool{
		"ssl-default-protocol": true, "starttls-always": true, "starttls-auto": true,
		"starttls-auto-unsupported": false, "never-with-secure-socket": false, "never": false,
	} {
		if gotTLS, ok := tlsBySubject[subject]; !ok || gotTLS != wantTLS {
			t.Errorf("%s: received=%v tls=%v, want tls=%v", subject, ok, gotTLS, wantTLS)
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
	closedPort := newFakeSmtpServer(t, smtpPlain, nil, nil)
	closedPort.close()

	runEmailCase(t, "email-smtp-server-errors-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;

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

    email:SmtpClient legacyAuth = check smtpClient(%[7]s);
    email:Error? legacy = legacyAuth->send("bob@localhost", "s", "alice@localhost", "b");
    report("legacy-auth-login", legacy); // @output legacy-auth-login: [sent]

    // A dropped connection is an I/O failure, not a rejected recipient.
    email:SmtpClient dropping = check smtpClient(%[8]s);
    email:Error? dropped = dropping->send("bob@localhost", "s", "alice@localhost", "b");
    report("dropped-after-mail", dropped); // @output dropped-after-mail: [Exception reading response]

    // Non-ASCII addresses need the SMTPUTF8 extension; jBallerina sends them without it.
    email:Error? utf8Unsupported = plainAuth->send("jösé@localhost", "s", "alice@localhost", "b");
    report("utf8-unsupported", utf8Unsupported); // @output utf8-unsupported: [SMTPUTF8 is required for non-ASCII addresses but host does not support SMTPUTF8]
    email:SmtpClient utf8Server = check smtpClient(%[9]s);
    email:Error? utf8 = utf8Server->send("jösé@localhost", "s", "alice@localhost", "b");
    report("utf8", utf8); // @output utf8: [sent]

    email:SmtpClient closed = check smtpClient(%[6]s);
    email:Error? refused = closed->send("bob@localhost", "s", "alice@localhost", "b");
    report("refused", refused); // @output refused: [Could not connect to SMTP host: 127.0.0.1]
    io:println(refused is email:Error && refused.cause() is error); // @output true
}
`, rejecting.port(), rejectData.port(), plainAuth.port(), cramOnly.port(), noAuth.port(), closedPort.port(), legacyAuth.port(), dropAfterMail.port(), utf8Server.port()))

	if got := utf8Server.received(); len(got) != 1 || got[0].mailParams != "SMTPUTF8" || !slices.Equal(got[0].recipients, []string{"jösé@localhost"}) {
		t.Errorf("SMTPUTF8 server: received %d messages, want one sent with SMTPUTF8 to jösé@localhost", len(got))
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
	runEmailCase(t, "email-smtp-transfer-encodings-v", fmt.Sprintf(`
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

    check sevenBit->send("bob@localhost", "quoted-comma", "alice@localhost", "Body",
        headers = {"To": "\"Doe, Jane\" <jane@localhost>, localuser"});
    io:println("sent"); // @output sent
}
`, eightBit.port(), sevenBit.port()))

	got := messagesBySubject(t, append(eightBit.received(), sevenBit.received()...))
	expect := map[string]string{
		"8bitmime":    `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | text/plain; charset=UTF-8(8bit)"café"]`,
		"no-8bitmime": `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | text/plain; charset=UTF-8(quoted-printable)"café"]`,
		"qp-binary":   `multipart/mixed[text/plain; charset=us-ascii(7bit)"Body" | image/png(quoted-printable)"A\nB\rC"]`,
		// UTF-16 output is binary data, so its 0x0A byte is not rewritten as a line ending.
		"utf-16":       `text/plain; charset=UTF-16BE(base64)"N\n\x00x"`,
		"quoted-comma": `text/plain; charset=us-ascii(7bit)"Body"`,
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
	if params := got["8bitmime"].mailParams; params != "BODY=8BITMIME" {
		t.Errorf("8bitmime: MAIL parameters %q, want BODY=8BITMIME", params)
	}
	if r := got["no-8bitmime"]; r.mailParams != "" || strings.ContainsFunc(r.data, func(c rune) bool { return c >= 0x80 }) {
		t.Errorf("no-8bitmime: MAIL parameters %q, want none and no 8-bit data:\n%s", r.mailParams, r.data)
	}
	if data := got["qp-binary"].data; !strings.Contains(data, "A=0AB=0DC") {
		t.Errorf("qp-binary: line-break bytes are not encoded:\n%s", data)
	}
	if rcpts := got["quoted-comma"].recipients; !slices.Equal(rcpts, []string{"jane@localhost", "localuser"}) {
		t.Errorf("quoted-comma: recipients %q", rcpts)
	}
}
