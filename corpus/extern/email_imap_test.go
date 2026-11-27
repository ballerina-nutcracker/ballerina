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
	"crypto/tls"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
)

func TestEmailImapReceive(t *testing.T) {
	t.Parallel()
	message := func(subject, body string) string {
		return "From: alice@localhost\nTo: bob@localhost\nSubject: " + subject + "\n\n" + body + "\n"
	}
	server := newFakeImapServer(t, nil, false, map[string][]imapMessage{
		"INBOX": {
			{data: message("one", "first")},
			{data: message("trashed", "flagged by another client"), flags: []imap.Flag{imap.FlagSeen, imap.FlagDeleted}},
			{data: message("deleted unseen", "flagged by another client"), flags: []imap.Flag{imap.FlagDeleted}},
			{data: message("two", "second")},
			{data: "From: alice@localhost\nTo: bob@localhost\nSubject: odd encoding\nContent-Type: multipart/mixed; boundary=B\n\n" +
				"--B\nContent-Type: text/plain\n\nhello\n--B\nContent-Type: application/octet-stream\nContent-Transfer-Encoding: x-unknown\n\ndata\n--B--\n"},
		},
		"Archive":       {{data: message("archived", "old")}},
		readOnlyMailbox: {{data: message("read only", "x")}},
	}, imap.CapESearch)

	runEmailCase(t, "email-imap-receive-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/mime;
%[2]s
function imapClient(string password = "secret", int port = %[1]s) returns email:ImapClient|email:Error =>
    new ("127.0.0.1", "bob", password, port = port, security = email:START_TLS_NEVER);

function read(email:ImapClient imap, string folder = "INBOX") returns string {
    email:Message|email:Error? m = imap->receiveMessage(folder, timeout = 0);
    return m is email:Message ? m.subject : m is email:Error ? "error: " + m.message() : "none";
}

public function main() returns error? {
    email:ImapClient imap = check imapClient();
    // jBallerina's CLOSE after each read expunges "trashed", which another client flagged \Deleted (ballerina-library#9772).
    // Messages flagged \Deleted are skipped.
    io:println("1: ", read(imap)); // @output 1: one
    io:println("2: ", read(imap)); // @output 2: two
    // jBallerina fails on the unknown transfer encoding after setting \Seen, so the message is lost (ballerina-library#9773).
    email:Message|email:Error? odd = imap->receiveMessage(timeout = 0);
    check describe(odd);
    // @output odd encoding: to=bob@localhost cc=[] bcc=[] replyTo=alice@localhost from=alice@localhost sender= contentType=multipart/mixed; boundary=B
    // @output > body=hello htmlBody=<none>
    // @output > attachment application/octet-stream = [100,97,116,97]
    io:println("4: ", read(imap)); // @output 4: none
    io:println("lowercase inbox: ", read(imap, "inbox")); // @output lowercase inbox: none
    io:println("archive: ", read(imap, "Archive")); // @output archive: archived
    io:println("missing: ", read(imap, "NoSuchFolder")); // @output missing: error: NoSuchFolder not found
    io:println("wildcard: ", read(imap, "Arch*")); // @output wildcard: error: Arch* not found
    io:println("read-only: ", read(imap, "ReadOnly")); // @output read-only: error: Cannot open in desired mode
    check imap->close();
    // jBallerina panics with IllegalStateException (ballerina-library#9774).
    io:println("after close: ", read(imap)); // @output after close: error: Not connected

    email:ImapClient unused = check imapClient();
    email:Error? closed = unused->close();
    io:println("close without read: ", closed is ()); // @output close without read: true

    email:ImapClient|email:Error badPassword = imapClient("wrong");
    io:println(badPassword is email:Error ? badPassword.message() : "connected"); // @output Error while connecting to the IMAP store: [AUTHENTICATIONFAILED] Authentication failed
}
`, server.port(), popDescribe))

	if messages, unseen := server.status(t, "INBOX"); messages != 5 || unseen != 1 {
		t.Errorf("INBOX: %d messages, %d unseen; want all 5 kept and only the \\Deleted one unseen", messages, unseen)
	}
	commands := server.commands()
	if slices.ContainsFunc(commands, func(c string) bool { return c == "CLOSE" || c == "EXPUNGE" }) {
		t.Errorf("commands %q expunge the folder", commands)
	}
	if !slices.ContainsFunc(commands, func(c string) bool { return strings.Contains(c, "BODY.PEEK[]") }) {
		t.Errorf("commands %q do not fetch with BODY.PEEK", commands)
	}
}

func TestEmailImapSessions(t *testing.T) {
	t.Parallel()
	late := newFakeImapServer(t, nil, false, map[string][]imapMessage{"INBOX": nil})
	shared := newFakeImapServer(t, nil, false, map[string][]imapMessage{"INBOX": {{data: "From: alice@localhost\nTo: bob@localhost\nSubject: once\n\nx\n"}}})
	idle := newFakeImapServer(t, nil, false, map[string][]imapMessage{"INBOX": nil})
	closedPort := newFakeImapServer(t, nil, false, map[string][]imapMessage{"INBOX": nil})
	_ = closedPort.ln.Close()
	go func() {
		time.Sleep(1500 * time.Millisecond)
		late.add(t, "INBOX", imapMessage{data: "From: alice@localhost\nTo: bob@localhost\nSubject: late\n\nx\n"})
	}()

	start := time.Now()
	runEmailCase(t, "email-imap-sessions-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/lang.runtime;

function imapClient(int port) returns email:ImapClient|email:Error =>
    new ("127.0.0.1", "bob", "secret", port = port, security = email:START_TLS_NEVER);

function read(email:ImapClient imap, decimal timeout) returns string {
    email:Message|email:Error? m = imap->receiveMessage(timeout = timeout);
    return m is email:Message ? m.subject : m is email:Error ? "error: " + m.message() : "none";
}

public function main() returns error? {
    email:ImapClient waiting = check imapClient(%[1]s);
    io:println("late: ", read(waiting, 30)); // @output late: late

    // Two concurrent reads on one client do not both return the one message.
    email:ImapClient both = check imapClient(%[4]s);
    future<string> other = start read(both, 0);
    string first = read(both, 0);
    string second = check wait other;
    boolean once = first == "once" && second == "none" || first == "none" && second == "once";
    io:println("concurrent reads return the message once: ", once); // @output concurrent reads return the message once: true

    email:ImapClient polling = check imapClient(%[2]s);
    future<string> pending = start read(polling, 30);
    runtime:sleep(0.5);
    check polling->close();
    string|error interrupted = wait pending;
    io:println("close while waiting: ", interrupted); // @output close while waiting: error: Not connected

    email:ImapClient|email:Error refused = imapClient(%[3]s);
    if refused is email:Error {
        string message = refused.message();
        io:println(message.substring(0, <int>message.indexOf(", port:")), " ", refused.cause() is error); // @output Error while connecting to the IMAP store: Couldn't connect to host true
    }
}
`, late.port(), idle.port(), closedPort.port(), shared.port()))

	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("receiving took %v, want the late message returned at the next poll", elapsed)
	}
	// One login validates the client in init and one serves the whole waiting read.
	if logins := slices.DeleteFunc(late.commands(), func(c string) bool { return !strings.HasPrefix(c, "AUTHENTICATE") }); len(logins) != 2 {
		t.Errorf("late: %d logins, want 2", len(logins))
	}
}

func TestEmailImapTLS(t *testing.T) {
	t.Parallel()
	caCertPEM, serverCertPEM, serverKeyPEM, _, _ := generateTestCerts(t)
	serverCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}
	serverTLS := &tls.Config{Certificates: []tls.Certificate{serverCert}}
	inbox := func(subject string) map[string][]imapMessage {
		return map[string][]imapMessage{"INBOX": {{data: "From: a@localhost\nTo: b@localhost\nSubject: " + subject + "\n\nx\n"}}}
	}
	implicit := newFakeImapServer(t, serverTLS, true, inbox("implicit"))
	startTLS := newFakeImapServer(t, serverTLS, false, inbox("starttls"))
	plain := newFakeImapServer(t, nil, false, inbox("plain"))
	caFile := writeTempFile(t, "ca.pem", caCertPEM)

	runEmailCase(t, "email-imap-tls-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;

function read(string label, int port, email:Security security) {
    email:ImapClient|email:Error imap = new ("127.0.0.1", "bob", "secret", port = port, security = security,
        secureSocket = {cert: "%[4]s"});
    if imap is email:Error {
        io:println(label, ": ", imap.message());
        return;
    }
    email:Message|email:Error? m = imap->receiveMessage(timeout = 0);
    io:println(label, ": ", m is email:Message ? m.subject : m is email:Error ? m.message() : "none");
}

public function main() {
    read("ssl", %[1]s, email:SSL); // @output ssl: implicit
    // jBallerina opens implicit TLS for START_TLS_ALWAYS (ballerina-library#9729).
    read("starttls-always", %[2]s, email:START_TLS_ALWAYS); // @output starttls-always: starttls
    read("starttls-auto-unsupported", %[3]s, email:START_TLS_AUTO); // @output starttls-auto-unsupported: plain
    read("starttls-always-unsupported", %[3]s, email:START_TLS_ALWAYS); // @output starttls-always-unsupported: Error while connecting to the IMAP store: STARTTLS required but not supported by server
}
`, implicit.port(), startTLS.port(), plain.port(), caFile))

	if commands := startTLS.commands(); !slices.Contains(commands, "STARTTLS") {
		t.Errorf("STARTTLS server: no STARTTLS command in %q", commands)
	}
}

// TestEmailImapNegotiation checks the STARTTLS negotiation before the session starts against
// servers that answer with scripted replies.
func TestEmailImapNegotiation(t *testing.T) {
	t.Parallel()
	capabilities := "* CAPABILITY IMAP4rev1 STARTTLS\r\nS1 OK done\r\n"
	refused := newScriptedServer(t, "* OK ready\r\n", capabilities, "S2 NO not now\r\n")
	greetingCode := newScriptedServer(t, "* ok [capability imap4rev1 starttls] ready\r\n", "S2 no not now\r\n")
	untagged := newScriptedServer(t, "* ok ready\r\n", "* capability imap4rev1 starttls\r\nS1 ok done\r\n", "S2 no not now\r\n")
	taggedCode := newScriptedServer(t, "* OK ready\r\n", "S1 OK [CAPABILITY IMAP4rev1 STARTTLS] done\r\n", "S2 NO not now\r\n")
	longLine := newScriptedServer(t, "* OK "+strings.Repeat("x", 16<<10)+"\r\n")

	runEmailCase(t, "email-imap-negotiation-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;

function connect(string label, int port, email:Security security) {
    email:ImapClient|email:Error imap = new ("127.0.0.1", "bob", "secret", port = port, security = security);
    if imap is email:ImapClient {
        io:println(label, ": connected");
        return;
    }
    error? cause = imap.cause();
    io:println(label, ": ", imap.message(), cause is error ? " (" + cause.message() + ")" : "");
}

public function main() {
    // A refused STARTTLS fails the connection in both STARTTLS modes, as JavaMail does.
    connect("refused-auto", %[1]s, email:START_TLS_AUTO); // @output refused-auto: Error while connecting to the IMAP store: not now
    connect("refused-always", %[1]s, email:START_TLS_ALWAYS); // @output refused-always: Error while connecting to the IMAP store: not now
    // IMAP keywords are case-insensitive, wherever the server lists its capabilities.
    connect("greeting-code", %[2]s, email:START_TLS_ALWAYS); // @output greeting-code: Error while connecting to the IMAP store: not now
    connect("untagged", %[3]s, email:START_TLS_ALWAYS); // @output untagged: Error while connecting to the IMAP store: not now
    connect("tagged-code", %[4]s, email:START_TLS_ALWAYS); // @output tagged-code: Error while connecting to the IMAP store: not now
    // A line longer than 16 KiB before the session starts fails the connection.
    connect("long-line", %[5]s, email:START_TLS_AUTO); // @output long-line: Error while connecting to the IMAP store: Connect failed (line too long)
}
`, refused, greetingCode, untagged, taggedCode, longLine))
}
