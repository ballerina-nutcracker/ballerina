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
	"crypto/tls"
	"fmt"
	"slices"
	"testing"
	"time"
)

// popDescribe prints a received message compactly, with line breaks in bodies made visible.
const popDescribe = `
function escape(string s) returns string {
    string out = "";
    foreach string c in s {
        out += c == "\r" ? "\\r" : c == "\n" ? "\\n" : c;
    }
    return out;
}

function describe(email:Message|email:Error? m) returns error? {
    if m is () {
        io:println("none");
        return;
    }
    if m is email:Error {
        io:println("error: ", m.message());
        return;
    }
    io:println(m.subject, ": to=", m.to, " cc=", m?.cc, " bcc=", m?.bcc, " replyTo=", m?.replyTo,
        " from=", m?.'from, " sender=", m?.sender, " contentType=", m?.contentType);
    io:println("> body=", escape(m?.body ?: "<none>"), " htmlBody=", escape(m?.htmlBody ?: "<none>"));
    var attachments = m?.attachments;
    if attachments is mime:Entity[] {
        foreach mime:Entity a in attachments {
            check describeEntity(a, "> ");
        }
    }
}

function describeEntity(mime:Entity e, string indent) returns error? {
    string disposition = e.hasHeader("Content-Disposition") ? " (" + check e.getHeader("Content-Disposition") + ")" : "";
    mime:Entity[]|error parts = e.getBodyParts();
    if parts is mime:Entity[] && e.getContentType().indexOf("multipart/") == 0 {
        io:println(indent, "attachment ", e.getContentType(), disposition, ":");
        foreach mime:Entity p in parts {
            check describeEntity(p, indent + "> ");
        }
        return;
    }
    byte[] data = check e.getByteArray();
    io:println(indent, "attachment ", e.getContentType(), disposition, " = ", data);
}
`

func TestEmailPopReceive(t *testing.T) {
	t.Parallel()
	mailbox := []string{
		"From: Alice <alice@localhost>\nTo: bob@localhost\nSubject: plain\n\nline1\nline2\n",
		// A body inside multipart/mixed > multipart/alternative fills body and htmlBody (ballerina-library#9761).
		"From: alice@localhost\nTo: bob@localhost, Carol <carol@localhost>\nCc: dave@localhost\nSubject: mixed\nMIME-Version: 1.0\n" +
			"Content-Type: multipart/mixed; boundary=OUTER\n\npreamble\n--OUTER\nContent-Type: multipart/alternative; boundary=INNER\n\n" +
			"--INNER\nContent-Type: text/plain; charset=us-ascii\n\nplain text\n--INNER\nContent-Type: text/html; charset=us-ascii\n\n<p>html</p>\n--INNER--\n" +
			"--OUTER\nContent-Type: application/json\nContent-Disposition: attachment; filename=data.json\n\n{\"k\": 1}\n" +
			"--OUTER\nContent-Type: application/octet-stream\nContent-Transfer-Encoding: base64\n\nAAEC/w==\n--OUTER--\nepilogue\n",
		"From: alice@localhost\nTo: bob@localhost\nReply-To: replies@localhost\nSender: agent@localhost\nSubject: alternative\n" +
			"Content-Type: multipart/alternative; boundary=ALT\n\n--ALT\nContent-Type: text/plain\n\nplain only\n--ALT\nContent-Type: text/html\n\n<b>html only</b>\n--ALT--\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: html\nContent-Type: text/html; charset=UTF-8\n\n<p>only html</p>\n",
		// A text body is decoded in its charset (ballerina-library#9762).
		"From: alice@localhost\nTo: bob@localhost\nSubject: latin1\nContent-Type: text/csv; charset=ISO-8859-1\nContent-Transfer-Encoding: quoted-printable\n\ncaf=E9\n",
		"Received: from a\nReceived: from b\nFrom: alice@localhost\nTo: bob@localhost\nSubject: =?UTF-8?Q?gr=C3=BC=C3=9Fe?=\n" +
			"X-Folded: first\n second\nContent-Type: application/json\n\n{\"a\": 1}\n",
		// A broken multipart is delivered with its raw body instead of failing every read (ballerina-library#9760).
		"From: alice@localhost\nTo: bob@localhost\nSubject: broken\nContent-Type: multipart/mixed\n\nno boundary\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: pdf\nContent-Type: application/pdf\nContent-Transfer-Encoding: base64\n\nJVBERg==\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: related\nContent-Type: multipart/related; boundary=REL\n\n" +
			"--REL\nContent-Type: text/html\n\n<img src=\"cid:logo\">\n--REL\nContent-Type: image/png\nContent-ID: <logo>\nContent-Transfer-Encoding: base64\n\nAQID\n--REL--\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: nested\nContent-Type: multipart/mixed; boundary=A\n\n" +
			"--A\nContent-Type: text/plain\n\nsee parts\n--A\nContent-Type: multipart/mixed; boundary=B\n\n--B\n\nno type\n--B\nContent-Type: text/plain; name=x.txt\nContent-Disposition: attachment\n\nx\n--B--\n--A--\n",
		"From: =?UTF-8?Q?J=C3=BCrgen?= <juergen@localhost>\nTo: \"Doe, John\" <john@localhost>\nSubject: names\n\nhi\n",
		"From: alice@localhost\nTo: undisclosed-recipients:;\nSubject: no recipients\n\nhi\n",
		// An address list that does not parse is returned as written.
		"From: alice@localhost\nTo: bob@localhost, <broken\nSubject: =?ISO-8859-1?Q?caf=E9?=\nContent-Type: text/plain; charset=x-unknown\n\nunknown charset\n",
		"From: alice@localhost\nTo: =?windows-1252?Q?Jos=E9?= <jose@localhost>, \"Doe, John\" <john@localhost>\nSubject: windows-1252\n\nhi\n",
		// The body is the first text part, even after an attachment whose filename is not quoted.
		"From: alice@localhost\nTo: bob@localhost\nSubject: leading attachment\nContent-Type: multipart/mixed; boundary=L\n\n" +
			"--L\nContent-Type: application/pdf\nContent-Disposition: attachment; filename=my report.pdf\n\n%PDF\n--L\nContent-Type: text/plain\n\nbody after\n--L--\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: unpadded\nContent-Transfer-Encoding: base64\n\naGVsbG8\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: attached multipart\nContent-Type: multipart/alternative; boundary=X\n\n" +
			"--X\nContent-Type: text/plain\n\nouter\n--X\nContent-Type: multipart/mixed; boundary=Y\nContent-Disposition: attachment; filename=fwd\n\n--Y\nContent-Type: text/plain\n\ninner\n--Y--\n--X--\n",
		"",
		"From: alice@localhost\nTo: bob@localhost\nSubject: text and html\nContent-Type: multipart/mixed; boundary=M\n\n" +
			"--M\nContent-Type: text/plain\n\nplain part\n--M\nContent-Type: text/html\n\n<i>html part</i>\n--M--\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: text after related\nContent-Type: multipart/mixed; boundary=M\n\n" +
			"--M\nContent-Type: multipart/related; boundary=R\n\n--R\nContent-Type: image/png\n\npng\n--R--\n--M\nContent-Type: text/plain\n\nlate text\n--M--\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: csv attachment\nContent-Type: text/csv\nContent-Disposition: attachment; filename=r.csv\n\na,b\n",
		"From: alice@localhost\nTo: \"Doe, John\" <john@localhost>, bad@@addr\nSubject: bad address\n\nhi\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: broken nested\nContent-Type: multipart/mixed; boundary=M\n\n" +
			"--M\nContent-Type: text/plain\n\ntext\n--M\nContent-Type: multipart/mixed\n\nraw\n--M--\n",
		"From: alice@localhost\nTo: bob@localhost\nSubject: truncated\nContent-Type: multipart/mixed; boundary=M\n\n--M\nContent-Type: text/plain\n\ncut short\n--M\n",
		// Header bytes that are not UTF-8 are read as ISO-8859-1.
		"From: alice@localhost\nTo: bob@localhost\nSubject: caf\xe9 latin1\n\nhi\n",
		// jBallerina reads UTF-8 header bytes as ISO-8859-1 (ballerina-library#9806).
		"From: alice@localhost\nTo: bob@localhost\nSubject: caf\xc3\xa9 utf8\n\nhi\n",
	}
	server := newFakePopServer(t, smtpPlain, nil, mailbox, nil)

	runEmailCase(t, "email-pop-receive-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/mime;
%[2]s
public function main() returns error? {
    email:PopClient pop = check new ("127.0.0.1", "bob", "secret", port = %[1]s, security = email:START_TLS_NEVER);
    foreach int _ in 0 ..< %[3]d {
        email:Message|email:Error? m = pop->receiveMessage(timeout = 0);
        check describe(m);
    }
    // @output plain: to=bob@localhost cc=[] bcc=[] replyTo=Alice <alice@localhost> from=Alice <alice@localhost> sender= contentType=text/plain
    // @output > body=line1\r\nline2\r\n htmlBody=<none>
    // @output mixed: to=["bob@localhost","Carol <carol@localhost>"] cc=dave@localhost bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed; boundary=OUTER
    // @output > body=plain text htmlBody=<p>html</p>
    // @output > attachment application/json (attachment; filename=data.json) = [123,34,107,34,58,32,49,125]
    // @output > attachment application/octet-stream = [0,1,2,255]
    // @output alternative: to=bob@localhost cc=[] bcc=[] replyTo=replies@localhost from=alice@localhost sender=agent@localhost contentType=multipart/alternative; boundary=ALT
    // @output > body=plain only htmlBody=<b>html only</b>
    // @output html: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/html; charset=UTF-8
    // @output > body= htmlBody=<p>only html</p>\r\n
    // @output latin1: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/csv; charset=ISO-8859-1
    // @output > body=café\r\n htmlBody=<none>
    // @output grüße: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=application/json
    // @output > body={"a": 1}\r\n htmlBody=<none>
    // @output broken: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed
    // @output > body=no boundary\r\n htmlBody=<none>
    // @output pdf: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=application/pdf
    // @output > body= htmlBody=<none>
    // @output > attachment application/pdf = [37,80,68,70]
    // @output related: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/related; boundary=REL
    // @output > body= htmlBody=<img src="cid:logo">
    // @output > attachment image/png = [1,2,3]
    // @output nested: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed; boundary=A
    // @output > body=see parts htmlBody=<none>
    // @output > attachment multipart/mixed; boundary=B:
    // @output > > attachment text/plain = [110,111,32,116,121,112,101]
    // @output > > attachment text/plain; name=x.txt (attachment) = [120]
    // @output names: to="Doe, John" <john@localhost> cc=[] bcc=[] replyTo==?UTF-8?q?J=C3=BCrgen?= <juergen@localhost> from==?UTF-8?q?J=C3=BCrgen?= <juergen@localhost> sender= contentType=text/plain
    // @output > body=hi\r\n htmlBody=<none>
    // @output no recipients: to=[] cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/plain
    // @output > body=hi\r\n htmlBody=<none>
    // @output café: to=["bob@localhost","<broken"] cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/plain; charset=x-unknown
    // @output > body=unknown charset\r\n htmlBody=<none>
    // @output windows-1252: to=["=?UTF-8?q?Jos=C3=A9?= <jose@localhost>","\"Doe, John\" <john@localhost>"] cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/plain
    // @output > body=hi\r\n htmlBody=<none>
    // @output leading attachment: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed; boundary=L
    // @output > body=body after htmlBody=<none>
    // @output > attachment application/pdf (attachment; filename=my report.pdf) = [37,80,68,70]
    // @output unpadded: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/plain
    // @output > body=hello htmlBody=<none>
    // @output attached multipart: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/alternative; boundary=X
    // @output > body=outer htmlBody=<none>
    // @output > attachment multipart/mixed; boundary=Y (attachment; filename=fwd):
    // @output > > attachment text/plain = [105,110,110,101,114]
    // @output : to=[] cc=[] bcc=[] replyTo=[] from= sender= contentType=text/plain
    // @output > body= htmlBody=<none>
    // @output text and html: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed; boundary=M
    // @output > body=plain part htmlBody=<i>html part</i>
    // @output text after related: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed; boundary=M
    // @output > body=late text htmlBody=<none>
    // @output > attachment multipart/related; boundary=R:
    // @output > > attachment image/png = [112,110,103]
    // @output csv attachment: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/csv
    // @output > body= htmlBody=<none>
    // @output > attachment text/csv (attachment; filename=r.csv) = [97,44,98,13,10]
    // @output bad address: to=["\"Doe, John\" <john@localhost>","bad@@addr"] cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/plain
    // @output > body=hi\r\n htmlBody=<none>
    // @output broken nested: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed; boundary=M
    // @output > body=text htmlBody=<none>
    // @output > attachment application/octet-stream = [114,97,119]
    // @output truncated: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed; boundary=M
    // @output > body=cut short htmlBody=<none>
    // @output café latin1: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/plain
    // @output > body=hi\r\n htmlBody=<none>
    // @output café utf8: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=text/plain
    // @output > body=hi\r\n htmlBody=<none>
    email:Message|email:Error? last = pop->receiveMessage(timeout = 0);
    check describe(last); // @output none
    check pop->close();
}
`, server.port(), popDescribe, len(mailbox)))

	if n := server.remaining(); n != 0 {
		t.Errorf("%d messages left on the server, want every received message deleted", n)
	}
}

func TestEmailPopSessions(t *testing.T) {
	t.Parallel()
	once := newFakePopServer(t, smtpPlain, nil, []string{"From: a@localhost\nTo: b@localhost\nSubject: once\n\nx\n"}, nil)
	late := newFakePopServer(t, smtpPlain, nil, []string{"From: a@localhost\nTo: b@localhost\nSubject: late\n\nx\n"},
		func(s *fakePopServer) { s.hiddenSessions = 2 })
	empty := newFakePopServer(t, smtpPlain, nil, nil, nil)
	failRetr := newFakePopServer(t, smtpPlain, nil, []string{"Subject: locked\n\nx\n"}, func(s *fakePopServer) { s.failRetr = true })
	closedPort := newFakePopServer(t, smtpPlain, nil, nil, nil)
	closedPort.close()
	backoff := newFakePopServer(t, smtpPlain, nil, nil, nil)
	polling := newFakePopServer(t, smtpPlain, nil, nil, nil)
	readOnly := newFakePopServer(t, smtpPlain, nil, []string{"Subject: kept\n\nx\n"}, func(s *fakePopServer) { s.failDele = true })

	start := time.Now()
	runEmailCase(t, "email-pop-sessions-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/lang.runtime;

function popClient(int port, string password = "secret") returns email:PopClient|email:Error =>
    new ("127.0.0.1", "bob", password, port = port, security = email:START_TLS_NEVER);

function read(email:PopClient pop, string folder = "INBOX", decimal timeout = 0) returns string {
    email:Message|email:Error? m = pop->receiveMessage(folder, timeout);
    return m is email:Message ? m.subject : m is email:Error ? "error: " + m.message() : "none";
}

public function main() returns error? {
    // jBallerina panics when close is called before any read (ballerina-library#9758).
    email:PopClient unused = check popClient(%[1]s);
    email:Error? closed = unused->close();
    io:println("close without read: ", closed is ()); // @output close without read: true

    // jBallerina never deletes the last message read, so the next session receives it again (ballerina-library#9759).
    email:PopClient first = check popClient(%[1]s);
    io:println("first: ", read(first)); // @output first: once
    check first->close();
    email:PopClient second = check popClient(%[1]s);
    io:println("second: ", read(second)); // @output second: none

    // jBallerina polls every 100 seconds, so a message arriving during the wait is returned only at the timeout (ballerina-library#9757).
    email:PopClient waiting = check popClient(%[2]s);
    io:println("late: ", read(waiting, timeout = 60)); // @output late: late
    email:PopClient idle = check popClient(%[3]s);
    io:println("sub-second: ", read(idle, timeout = 0.5)); // @output sub-second: none

    io:println("other folder: ", read(idle, "Sent")); // @output other folder: error: folder is not INBOX
    io:println("lowercase inbox: ", read(idle, "inbox")); // @output lowercase inbox: none
    check idle->close();
    io:println("after close: ", read(idle)); // @output after close: error: Not connected

    email:PopClient slow = check popClient(%[6]s);
    io:println("backoff: ", read(slow, timeout = 3.5)); // @output backoff: none
    email:PopClient waiting2 = check popClient(%[7]s);
    future<string> pending = start read(waiting2, timeout = 30);
    runtime:sleep(0.5);
    check waiting2->close();
    string|error interrupted = wait pending;
    io:println("close while waiting: ", interrupted); // @output close while waiting: error: Not connected

    email:PopClient locked = check popClient(%[5]s);
    io:println("rejected retrieve: ", read(locked)); // @output rejected retrieve: error: message locked

    // A message the server refuses to delete is still delivered, and stays for the next read.
    email:PopClient readOnly = check popClient(%[8]s);
    io:println("rejected delete: ", read(readOnly)); // @output rejected delete: kept
    io:println("rejected delete again: ", read(readOnly)); // @output rejected delete again: kept

    email:PopClient|email:Error badPassword = popClient(%[3]s, "wrong");
    io:println(badPassword is email:Error ? badPassword.message() : "connected"); // @output Error while connecting to the POP3 store: [AUTH] Authentication failed
    email:PopClient|email:Error refused = popClient(%[4]s);
    if refused is email:Error {
        string message = refused.message();
        io:println(message.substring(0, <int>message.indexOf(", port:")), " ", refused.cause() is error); // @output Error while connecting to the POP3 store: Couldn't connect to host true
    }
}
`, once.port(), late.port(), empty.port(), closedPort.port(), failRetr.port(), backoff.port(), polling.port(), readOnly.port()))

	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("receiving took %v, want the late message returned at the next poll", elapsed)
	}
	if got := once.remaining(); got != 0 {
		t.Errorf("once: %d messages left, want the received message deleted at close", got)
	}
	// Reads at the start and, unless the first read outlasts the wait, at the end of the 0.5
	// second wait, and one for the lowercase folder.
	if n := statCount(empty); n < 2 || n > 3 {
		t.Errorf("empty mailbox: %d STAT commands, want 2 or 3", n)
	}
	// Reads after waits of 1, 2 and 0.5 seconds, instead of one every second.
	if n := statCount(backoff); n < 3 || n > 4 {
		t.Errorf("backoff: %d STAT commands, want 3 or 4", n)
	}
	if n := statCount(polling); n > 2 {
		t.Errorf("close while waiting: %d STAT commands, want polling to stop at close", n)
	}
}

func statCount(server *fakePopServer) int {
	n := 0
	for _, command := range server.log() {
		if command == "STAT" {
			n++
		}
	}
	return n
}

func TestEmailPopTLS(t *testing.T) {
	t.Parallel()
	caCertPEM, serverCertPEM, serverKeyPEM, _, _ := generateTestCerts(t)
	serverCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}
	serverTLS := &tls.Config{Certificates: []tls.Certificate{serverCert}}
	message := func(subject string) []string {
		return []string{"From: a@localhost\nTo: b@localhost\nSubject: " + subject + "\n\nx\n"}
	}
	implicit := newFakePopServer(t, smtpImplicitTLS, serverTLS, message("implicit"), nil)
	startTLS := newFakePopServer(t, smtpStartTLS, serverTLS, message("stls"), nil)
	plain := newFakePopServer(t, smtpPlain, nil, message("plain"), nil)
	refusing := newFakePopServer(t, smtpStartTLS, serverTLS, message("refused"), func(s *fakePopServer) { s.refuseSTLS = true })
	dropping := newFakePopServer(t, smtpStartTLS, serverTLS, nil, func(s *fakePopServer) { s.dropCAPA = true })
	caFile := writeTempFile(t, "ca.pem", caCertPEM)

	runEmailCase(t, "email-pop-tls-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;

function read(string label, int port, email:Security security) {
    email:PopClient|email:Error pop = new ("127.0.0.1", "bob", "secret", port = port, security = security,
        secureSocket = {cert: "%[4]s"});
    if pop is email:Error {
        io:println(label, ": ", pop.message());
        return;
    }
    email:Message|email:Error? m = pop->receiveMessage(timeout = 0);
    io:println(label, ": ", m is email:Message ? m.subject : m is email:Error ? m.message() : "none");
}

public function main() {
    read("ssl", %[1]s, email:SSL); // @output ssl: implicit
    read("starttls-always", %[2]s, email:START_TLS_ALWAYS); // @output starttls-always: stls
    read("starttls-auto-unsupported", %[3]s, email:START_TLS_AUTO); // @output starttls-auto-unsupported: plain
    read("starttls-always-unsupported", %[3]s, email:START_TLS_ALWAYS); // @output starttls-always-unsupported: Error while connecting to the POP3 store: STLS is required but host does not support STLS
    read("starttls-auto-refused", %[5]s, email:START_TLS_AUTO); // @output starttls-auto-refused: refused
    read("starttls-always-refused", %[5]s, email:START_TLS_ALWAYS); // @output starttls-always-refused: Error while connecting to the POP3 store: TLS not available
    read("capa-dropped", %[6]s, email:START_TLS_AUTO); // @output capa-dropped: Error while connecting to the POP3 store: Connect failed
}
`, implicit.port(), startTLS.port(), plain.port(), caFile, refusing.port(), dropping.port()))

	if log := startTLS.log(); !slices.Contains(log, "STLS") {
		t.Errorf("STARTTLS server: no STLS command in %q", log)
	}
}
