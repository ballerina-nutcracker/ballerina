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

import ballerina/io;
import ballerina/lang.runtime;
import ballerina/messaging;

type Entry record {|
    string id;
    anydata payload;
    boolean taken = false;
|};

isolated string[] events = [];

isolated function note(string event) {
    lock {
        events.push(event);
    }
}

isolated function eventCount() returns int {
    lock {
        return events.length();
    }
}

isolated function takeEvents(int expected) returns string {
    foreach int _ in 0 ..< 250 {
        if eventCount() >= expected {
            break;
        }
        runtime:sleep(0.02);
    }
    lock {
        string joined = "";
        foreach string event in events {
            joined += joined == "" ? event : ", " + event;
        }
        events = [];
        return joined;
    }
}

isolated client class FixedStore {
    *messaging:Store;

    private Entry[] entries = [];
    private final string name;
    private final boolean failStore;
    private final boolean failRetrieve;
    private final boolean failAck;
    private boolean retrieveFailed = false;

    isolated function init(string name, boolean failStore = false, boolean failRetrieve = false,
            boolean failAck = false) {
        self.name = name;
        self.failStore = failStore;
        self.failRetrieve = failRetrieve;
        self.failAck = failAck;
    }

    isolated remote function store(anydata payload) returns error? {
        if self.failStore {
            note(self.name + " store failed");
            return error("store failed");
        }
        lock {
            self.entries.push({id: self.name + (self.entries.length() + 1).toString(), payload: payload.clone()});
        }
        note(self.name + " stored " + payload.toString());
    }

    isolated remote function retrieve() returns messaging:Message|error? {
        if self.failRetrieve {
            boolean first;
            lock {
                first = !self.retrieveFailed;
                self.retrieveFailed = true;
            }
            if first {
                note(self.name + " retrieve failed");
                return error("retrieve failed");
            }
        }
        lock {
            foreach Entry entry in self.entries {
                if !entry.taken {
                    entry.taken = true;
                    return {id: entry.id, payload: entry.payload.clone()};
                }
            }
            return;
        }
    }

    isolated remote function acknowledge(string id, boolean success = true) returns error? {
        note("ack " + id + " " + success.toString());
        if self.failAck {
            return error("ack failed");
        }
    }
}

isolated map<int> attempts = {};

isolated function attemptsOf(string key) returns int {
    lock {
        return attempts[key] ?: 0;
    }
}

isolated service class Flaky {
    *messaging:StoreService;

    private final int failures;
    private final int panics;

    isolated function init(int failures, int panics = 0) {
        self.failures = failures;
        self.panics = panics;
    }

    isolated remote function onMessage(anydata payload) returns error? {
        string key = payload.toString();
        int attempt;
        lock {
            attempt = (attempts[key] ?: 0) + 1;
            attempts[key] = attempt;
        }
        if attempt <= self.panics {
            panic error("panicked on " + key + " attempt " + attempt.toString());
        }
        if attempt <= self.failures {
            return error("failed " + key + " attempt " + attempt.toString());
        }
        note("processed " + key);
    }
}

// Stops its listener from inside onMessage, so the stop always comes before the retry.
isolated service class StopsListener {
    *messaging:StoreService;

    private final messaging:StoreListener l;

    isolated function init(messaging:StoreListener l) {
        self.l = l;
    }

    isolated remote function onMessage(anydata payload) returns error? {
        string key = payload.toString();
        int attempt;
        lock {
            attempt = (attempts[key] ?: 0) + 1;
            attempts[key] = attempt;
        }
        lock {
            // A final isolated-object field still needs a lock: https://github.com/ballerina-nutcracker/ballerina/issues/1431
            check self.l.immediateStop();
        }
        return error("failed " + key + " attempt " + attempt.toString());
    }
}

// A failed acknowledge or retrieve is logged after the poll run's last store call, so no event follows the log.
isolated function awaitTrailingLog() {
    runtime:sleep(0.5);
}

function runOnce(messaging:StoreListener l, messaging:Store store, Flaky svc, string payload,
        int expectedEvents) returns string|error {
    check l.attach(svc);
    check store->store(payload);
    _ = takeEvents(1);
    check l.'start();
    string result = takeEvents(expectedEvents);
    check l.immediateStop();
    return result;
}

public function main() returns error? {
    FixedStore a = new ("a");
    messaging:StoreListener recovering = check new (a, pollingInterval = 0.02, retryInterval = 0.02);
    io:println(check runOnce(recovering, a, new (2), "recover", 2)); // @output processed recover, ack a1 true

    FixedStore b = new ("b");
    messaging:StoreListener noRetry = check new (b, pollingInterval = 0.02, maxRetries = 0,
        ackWithFailureAfterMaxRetries = false);
    io:println(check runOnce(noRetry, b, new (1), "noRetry", 1)); // @output ack b1 true

    FixedStore c = new ("c");
    messaging:StoreListener exhausted = check new (c, pollingInterval = 0.02, maxRetries = 1, retryInterval = 0.02);
    io:println(check runOnce(exhausted, c, new (5), "exhausted", 1)); // @output ack c1 false

    FixedStore d = new ("d");
    messaging:InMemoryMessageStore dead = new;
    messaging:StoreListener toDeadLetter = check new (d, pollingInterval = 0.02, maxRetries = 1,
        retryInterval = 0.02, deadLetterStore = dead);
    io:println(check runOnce(toDeadLetter, d, new (5), "toDeadLetter", 1)); // @output ack d1 true
    messaging:Message? deadLetter = dead->retrieve();
    io:println(deadLetter is messaging:Message ? deadLetter.payload : ()); // @output toDeadLetter

    FixedStore e = new ("e");
    FixedStore brokenDead = new ("dead", failStore = true);
    messaging:StoreListener deadLetterFails = check new (e, pollingInterval = 0.02, maxRetries = 0,
        deadLetterStore = brokenDead);
    io:println(check runOnce(deadLetterFails, e, new (5), "deadLetterFails", 2)); // @output dead store failed, ack e1 false

    FixedStore f = new ("f");
    messaging:StoreListener trapped = check new (f, pollingInterval = 0.02, retryInterval = 0.02);
    io:println(check runOnce(trapped, f, new (0, panics = 1), "trapped", 2)); // @output processed trapped, ack f1 true

    // A panic on a retry is trapped like one on the first attempt: https://github.com/ballerina-platform/ballerina-library/issues/9371
    FixedStore i = new ("i");
    messaging:StoreListener retryPanics = check new (i, pollingInterval = 0.02, retryInterval = 0.02);
    io:println(check runOnce(retryPanics, i, new (0, panics = 2), "retryPanics", 2)); // @output processed retryPanics, ack i1 true

    FixedStore j = new ("j");
    messaging:InMemoryMessageStore panicDead = new;
    messaging:StoreListener alwaysPanics = check new (j, pollingInterval = 0.02, maxRetries = 1, retryInterval = 0.02,
        deadLetterStore = panicDead);
    io:println(check runOnce(alwaysPanics, j, new (0, panics = 5), "alwaysPanics", 1)); // @output ack j1 true
    messaging:Message? panicked = panicDead->retrieve();
    io:println(panicked is messaging:Message ? panicked.payload : ()); // @output alwaysPanics

    FixedStore g = new ("g", failAck = true);
    messaging:StoreListener ackFails = check new (g, pollingInterval = 0.02);
    io:println(check runOnce(ackFails, g, new (0), "ackFails", 2)); // @output processed ackFails, ack g1 true
    awaitTrailingLog();

    // immediateStop cancels a message waiting to be retried: https://github.com/ballerina-platform/ballerina-library/issues/9705
    FixedStore k = new ("k");
    messaging:StoreListener stoppedMidRetry = check new (k, pollingInterval = 0.02, retryInterval = 0.3);
    check stoppedMidRetry.attach(new StopsListener(stoppedMidRetry));
    check k->store("stoppedMidRetry");
    _ = takeEvents(1);
    check stoppedMidRetry.'start();
    io:println(takeEvents(1)); // @output ack k1 false
    io:println(attemptsOf("stoppedMidRetry")); // @output 1

    FixedStore h = new ("h", failRetrieve = true);
    messaging:StoreListener retrieveFails = check new (h, pollingInterval = 0.5);
    check retrieveFails.attach(new Flaky(0));
    check retrieveFails.'start();
    io:println(takeEvents(1)); // @output h retrieve failed
    check retrieveFails.immediateStop();
    awaitTrailingLog();
}
