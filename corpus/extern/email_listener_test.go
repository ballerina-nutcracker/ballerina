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
	"fmt"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// listenerRecorder records the events of the services in a listener test, in dispatch order.
const listenerRecorder = `
isolated string[] events = [];

isolated function note(string event) {
    lock {
        events.push(event);
    }
}

// waitFor waits up to five seconds for count events and returns the events so far.
function waitFor(int count) returns string {
    foreach int _ in 0 ..< 100 {
        lock {
            if events.length() >= count {
                break;
            }
        }
        runtime:sleep(0.05);
    }
    lock {
        string joined = "";
        foreach string event in events {
            joined += (joined == "" ? "" : ", ") + event;
        }
        events.removeAll();
        return joined;
    }
}

service class Recorder {
    *email:Service;
    private final string name;

    function init(string name) {
        self.name = name;
    }

    remote function onMessage(email:Message message) {
        note(self.name + " " + message.subject);
    }

    remote function onError(email:Error err) {
        note(self.name + " error " + err.message());
    }

    remote function onClose(email:Error? err) {
        note(self.name + " closed" + (err is () ? "" : " " + err.message()));
    }
}
`

func imapText(subject string) imapMessage {
	return imapMessage{data: "From: alice@localhost\nTo: bob@localhost\nSubject: " + subject + "\n\nx\n"}
}

// imapServerExterns lets a test case add messages to server and stop it.
func imapServerExterns(t *testing.T, name string, server *fakeImapServer) []testharness.ExternRegistration {
	return []testharness.ExternRegistration{
		{Org: "$anon", Module: name, FuncName: "addMessage",
			Impl: func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
				server.add(t, "INBOX", imapText(args[0].(string)))
				return nil, nil
			}},
		{Org: "$anon", Module: name, FuncName: "stopServer",
			Impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
				server.stop()
				return nil, nil
			}},
	}
}

func TestEmailImapListenerServices(t *testing.T) {
	t.Parallel()
	server := newFakeImapServer(t, nil, false, map[string][]imapMessage{
		"INBOX": {imapText("m1"), imapText("m2"), imapText("m3")},
	})
	runEmailCase(t, "email-imap-listener-services-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/lang.runtime;
%[2]s
// The default interval would leave all but one message waiting if a poll read only one (ballerina-library#9783).
listener email:ImapListener mailbox = check new ({
    host: "127.0.0.1", username: "bob", password: "secret", port: %[1]s, security: email:START_TLS_NEVER
});

service "observer" on mailbox {
    remote function onMessage(email:Message message) {
        note("observer " + message.subject);
    }

    remote function onClose(email:Error? err) {
        io:println("observer closed: ", err is ());
    }
}

// jBallerina never registers a service attached with a path (ballerina-library#9777), and calls
// the onError and onClose methods this service does not declare (ballerina-library#9781).
service /inbox/watcher on mailbox {
    remote function onMessage(email:Message message) {
        note("watcher " + message.subject);
    }
}

public function testMain() {
    io:println(waitFor(6));
    // @output observer m1, watcher m1, observer m2, watcher m2, observer m3, watcher m3
}
// jBallerina never calls onClose (ballerina-library#9780).
// @output observer closed: true
`, server.port(), listenerRecorder))

	if messages, unseen := server.status(t, "INBOX"); messages != 3 || unseen != 0 {
		t.Errorf("INBOX: %d messages, %d unseen; want 3 seen", messages, unseen)
	}
}

func TestEmailImapListenerLifecycle(t *testing.T) {
	t.Parallel()
	server := newFakeImapServer(t, nil, false, map[string][]imapMessage{"INBOX": {imapText("m1")}})
	name := "email-imap-listener-lifecycle-v"
	runEmailCase(t, name, fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/lang.runtime;
%[2]s
function addMessage(string subject) = external;

function newListener(decimal pollingInterval = 0.1) returns email:ImapListener|email:Error =>
    new ({host: "127.0.0.1", username: "bob", password: "secret", port: %[1]s, security: email:START_TLS_NEVER,
        pollingInterval});

function startError(decimal pollingInterval) returns string {
    email:ImapListener|email:Error l = newListener(pollingInterval);
    if l is email:Error {
        return l.message();
    }
    error? started = l.'start();
    if started is error {
        return started.message();
    }
    error? stopped = l.gracefulStop();
    return stopped is error ? stopped.message() : "started";
}

service class ErrorsOnly {
    *email:Service;

    remote function onError(email:Error err) {
        note("error " + err.message());
    }
}

public function main() returns error? {
    // jBallerina panics (ballerina-library#9776).
    email:ImapListener|email:Error badPassword = new ({host: "127.0.0.1", username: "bob", password: "wrong",
        port: %[1]s, security: email:START_TLS_NEVER});
    io:println(badPassword is email:Error ? badPassword.message() : "created");
    // @output Error while connecting to the IMAP store: [AUTHENTICATIONFAILED] Authentication failed

    io:println(startError(0)); // @output Repeat Interval cannot be zero.
    io:println(startError(0.0001)); // @output Repeat Interval cannot be zero.
    io:println(startError(-1)); // @output Repeat interval must be >= 0
    io:println(startError(1e10)); // @output started
    io:println(startError(1e20)); // @output started
    io:println(startError(9.999999999999999999999999999999999E6144)); // @output started

    // jBallerina consumes messages while no service is attached (ballerina-library#9784).
    email:ImapListener idle = check newListener();
    check idle.'start();
    runtime:sleep(0.3);
    check idle.attach(new ErrorsOnly());
    runtime:sleep(0.3);
    check idle.gracefulStop();

    // jBallerina keeps only the last service of a class (ballerina-library#9778).
    email:ImapListener l = check newListener();
    Recorder first = new ("first");
    Recorder second = new ("second");
    check l.attach(first);
    check l.attach(second, "second");
    check l.attach(second);
    check l.'start();
    // jBallerina's second job keeps polling after the stop (ballerina-library#9782).
    check l.'start();
    io:println(waitFor(2)); // @output first m1, second m1

    // jBallerina's detach does nothing (ballerina-library#9779).
    check l.detach(second);
    addMessage("m2");
    io:println(waitFor(1)); // @output first m2

    check l.gracefulStop();
    check l.gracefulStop();
    addMessage("m3");
    runtime:sleep(0.3);
    io:println(waitFor(1)); // @output first closed

    check l.'start();
    io:println(waitFor(1)); // @output first m3
    check l.immediateStop();
    io:println(waitFor(1)); // @output first closed
}
`, server.port(), listenerRecorder), imapServerExterns(t, name, server)...)
}

func TestEmailImapListenerDispatch(t *testing.T) {
	t.Parallel()
	server := newFakeImapServer(t, nil, false, map[string][]imapMessage{"INBOX": nil})
	name := "email-imap-listener-dispatch-v"
	runEmailCase(t, name, fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/lang.runtime;
%[2]s
function addMessage(string subject) = external;

function stopServer() = external;

email:ImapListener l = check new ({host: "127.0.0.1", username: "bob", password: "secret", port: %[1]s,
    security: email:START_TLS_NEVER, pollingInterval: 0.1});

email:ImapListener restarting = check new ({host: "127.0.0.1", username: "bob", password: "secret", port: %[1]s,
    security: email:START_TLS_NEVER, pollingInterval: 0.1});

isolated boolean restarted = false;

service class Handler {
    *email:Service;

    remote function onMessage(email:Message message) {
        string subject = message.subject;
        if subject == "panic" {
            panic error("handler panicked");
        } else if subject == "slow" {
            note("started");
            runtime:sleep(0.5);
            note("finished");
        } else if subject == "stop" {
            error? stopped = l.gracefulStop();
            note(stopped is () ? "stopped from handler" : stopped.message());
        } else {
            note(subject);
        }
    }

    remote function onClose(email:Error? err) {
        note(err is () ? "closed" : err.message());
    }
}

isolated int stops = 0;

service class Restarter {
    *email:Service;

    remote function onMessage(email:Message message) {
        note(message.subject);
        if message.subject == "restart" {
            error? stopped = restarting.gracefulStop();
            note(stopped is () ? "stopping" : stopped.message());
        }
    }

    remote function onClose(email:Error? err) {
        boolean first;
        lock {
            first = !restarted;
            restarted = true;
        }
        if first && err is () {
            error? started = restarting.'start();
            note(started is () ? "restarted" : started.message());
        }
    }
}

service class ClosePanics {
    *email:Service;

    remote function onClose(email:Error? err) {
        if err is () {
            panic error("onClose panicked");
        }
    }
}

service class Silent {
    *email:Service;

    remote function onMessage(email:Message message) {
        _ = message;
    }
}

public function main() returns error? {
    check l.attach(new Handler());
    check l.'start();
    addMessage("panic");
    addMessage("after panic");
    io:println(waitFor(1)); // @output after panic

    // A graceful stop waits for the running handler; an immediate stop does not.
    addMessage("slow");
    _ = waitFor(1);
    check l.gracefulStop();
    io:println(waitFor(2)); // @output finished, closed
    // onClose follows the running handler, and a restart polls only after it.
    check l.'start();
    addMessage("slow");
    _ = waitFor(1);
    check l.immediateStop();
    check l.'start();
    addMessage("after restart");
    io:println(waitFor(3)); // @output finished, closed, after restart

    // A graceful stop after an immediate one waits for the running handler and onClose.
    addMessage("slow");
    _ = waitFor(1);
    check l.immediateStop();
    check l.gracefulStop();
    io:println(waitFor(0)); // @output finished, closed

    // A stop from a handler does not wait for the handler itself; the message still reaches the
    // other services before onClose.
    check l.attach(new Recorder("other"));
    check l.'start();
    addMessage("stop");
    runtime:sleep(0.5);
    io:println(waitFor(4)); // @output stopped from handler, other stop, closed, other closed

    // onClose can restart a listener that a handler stopped.
    check restarting.attach(new Restarter());
    check restarting.'start();
    addMessage("restart");
    io:println(waitFor(3)); // @output restart, stopping, restarted
    addMessage("after");
    io:println(waitFor(1)); // @output after
    check restarting.gracefulStop();

    // onClose runs on its own context, so its panic leaves the caller's locks held.
    email:ImapListener closing = check new ({host: "127.0.0.1", username: "bob", password: "secret",
        port: %[1]s, security: email:START_TLS_NEVER});
    check closing.attach(new ClosePanics());
    check closing.'start();
    lock {
        stops += 1;
        check closing.gracefulStop();
    }
    io:println("lock released"); // @output lock released

    email:ImapListener failing = check new ({host: "127.0.0.1", username: "bob", password: "secret",
        port: %[1]s, security: email:START_TLS_NEVER, pollingInterval: 0.1});
    check failing.attach(new Recorder("recorder"));
    check failing.attach(new Silent());
    stopServer();
    check failing.'start();
    string failure = waitFor(1);
    io:println(failure.substring(0, <int>failure.indexOf(", port:"))); // @output recorder error Couldn't connect to host
    check failing.gracefulStop();
}
`, server.port(), listenerRecorder), imapServerExterns(t, name, server)...)
}

func TestEmailPopListener(t *testing.T) {
	t.Parallel()
	text := func(subject string) string {
		return "From: alice@localhost\r\nTo: bob@localhost\r\nSubject: " + subject + "\r\n\r\nx\r\n"
	}
	server := newFakePopServer(t, smtpPlain, nil, []string{text("m1"), text("m2")}, nil)
	runEmailCase(t, "email-pop-listener-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/lang.runtime;
%[2]s
listener email:PopListener mailbox = check new ({
    host: "127.0.0.1", username: "bob", password: "secret", port: %[1]s, security: email:START_TLS_NEVER
});

service on mailbox {
    remote function onMessage(email:Message message) {
        note(message.subject);
    }

    remote function onClose(email:Error? err) {
        io:println("closed: ", err is ());
    }
}

public function testMain() returns error? {
    io:println(waitFor(2)); // @output m1, m2
    email:PopListener|email:Error badPassword = new ({host: "127.0.0.1", username: "bob", password: "wrong",
        port: %[1]s, security: email:START_TLS_NEVER});
    io:println(badPassword is email:Error ? badPassword.message() : "created");
    // @output Error while connecting to the POP3 store: [AUTH] Authentication failed
}
// @output closed: true
`, server.port(), listenerRecorder))

	if remaining := server.remaining(); remaining != 0 {
		t.Errorf("%d messages left on the server, want both deleted", remaining)
	}
}

func TestEmailPopListenerFailures(t *testing.T) {
	t.Parallel()
	text := func(subject string) string {
		return "From: alice@localhost\r\nTo: bob@localhost\r\nSubject: " + subject + "\r\n\r\nx\r\n"
	}
	undeletable := newFakePopServer(t, smtpPlain, nil, []string{text("kept")}, func(s *fakePopServer) { s.failDele = true })
	locked := newFakePopServer(t, smtpPlain, nil, []string{text("locked")}, func(s *fakePopServer) { s.failRetr = true })
	runEmailCase(t, "email-pop-listener-failures-v", fmt.Sprintf(`
import ballerina/email;
import ballerina/io;
import ballerina/lang.runtime;
%[3]s
function newListener(int port) returns email:PopListener|email:Error =>
    new ({host: "127.0.0.1", username: "bob", password: "secret", port, security: email:START_TLS_NEVER});

service class Failing {
    *email:Service;

    remote function onMessage(email:Message message) returns error? {
        note(message.subject);
        return error("handler failed");
    }
}

public function main() returns error? {
    // A message that cannot be deleted ends the poll instead of being read again at once.
    email:PopListener undeletable = check newListener(%[1]s);
    check undeletable.attach(new Failing());
    check undeletable.'start();
    runtime:sleep(0.5);
    io:println(waitFor(1)); // @output kept
    check undeletable.gracefulStop();

    // Without an onError method the read error is written to stderr.
    email:PopListener locked = check newListener(%[2]s);
    check locked.attach(new Failing());
    check locked.'start();
    runtime:sleep(0.3);
    check locked.gracefulStop();
    io:println(waitFor(0).length()); // @output 0
}
`, undeletable.port(), locked.port(), listenerRecorder))

	if remaining := undeletable.remaining(); remaining != 1 {
		t.Errorf("%d messages left on the undeletable server, want 1", remaining)
	}
}
