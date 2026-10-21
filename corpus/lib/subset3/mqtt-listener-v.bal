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

// Requires an MQTT v5 broker on mqtt:DEFAULT_URL (see corpus/lib/resources/mqtt/compose.yaml).
// Service methods report back over the "listener-v/events" topic so the output order is
// decided by a single stream rather than by thread scheduling.
import ballerina/io;
import ballerina/mqtt;

const EVENTS = "listener-v/events";

isolated int reports = 0;

isolated function report(string event) returns error? {
    int id;
    lock {
        reports += 1;
        id = reports;
    }
    mqtt:Client reporter = check new (mqtt:DEFAULT_URL, "listener-v-reporter-" + id.toString());
    _ = check reporter->publish(EVENTS, {payload: event.toBytes(), qos: 1});
    check reporter->disconnect();
    check reporter->close();
}

isolated service class Echo {
    *mqtt:Service;

    remote function onMessage(mqtt:Message message) returns error? {
        check report(string `echo ${check string:fromBytes(message.payload)} qos=${message.qos}`);
    }
}

isolated service class Responder {
    *mqtt:Service;

    remote function onMessage(mqtt:Message message, mqtt:Caller caller) returns error? {
        mqtt:Error? responded = caller->respond({payload: ("re: " + check string:fromBytes(message.payload)).toBytes()});
        check caller->complete();
        if responded is mqtt:Error {
            check report("respond failed: " + responded.message());
        }
    }

    remote function onComplete(mqtt:DeliveryToken token) returns error? {
        check report("onComplete " + token.topic);
    }
}

isolated service class OnlyOnError {
    *mqtt:Service;

    remote function onError(mqtt:Error err) returns error? {
        check report("onError " + err.message());
    }
}

isolated service class Panicking {
    *mqtt:Service;

    remote function onMessage(mqtt:Message message) returns error? {
        panic error("onMessage panicked on " + check string:fromBytes(message.payload));
    }

    remote function onError(mqtt:Error err) returns error? {
        check report("onError " + err.message());
    }
}

service class StoppingResponder {
    *mqtt:Service;
    private final mqtt:Listener owner;

    function init(mqtt:Listener owner) {
        self.owner = owner;
    }

    remote function onMessage(mqtt:Message message, mqtt:Caller caller) returns error? {
        check self.owner.immediateStop();
        mqtt:Error? responded = caller->respond({payload: message.payload});
        mqtt:Error? completed = caller->complete();
        check report(string `respond: ${responded is mqtt:Error ? responded.message() : "ok"}, complete: ${
            completed is mqtt:Error ? completed.message() : "ok"}`);
    }
}

function nextEvent(stream<mqtt:Message, error?> events) returns string|error {
    record {|mqtt:Message value;|}? next = check events.next();
    return next is () ? "end of stream" : string:fromBytes(next.value.payload);
}

public function main() returns error? {
    mqtt:Client c = check new (mqtt:DEFAULT_URL, "listener-v-client");
    check c->subscribe([EVENTS, "listener-v/replies"]);
    stream<mqtt:Message, error?> events = check c->receive();

    // Subscriptions given as a string[] use the default QoS 1.
    mqtt:Listener echo = check new (mqtt:DEFAULT_URL, "listener-v-echo", ["listener-v/echo1", "listener-v/echo2"]);
    check echo.attach(new Echo(), "ignored-name");
    check echo.'start();
    _ = check c->publish("listener-v/echo1", {payload: "a".toBytes(), qos: 2});
    io:println(check nextEvent(events)); // @output echo a qos=1
    _ = check c->publish("listener-v/echo2", {payload: "b".toBytes(), qos: 0});
    io:println(check nextEvent(events)); // @output echo b qos=0

    // Request/response through mqtt:Caller in manualAcks mode.
    mqtt:Listener responder = check new (mqtt:DEFAULT_URL, "listener-v-responder",
        [{topic: "listener-v/req", qos: 2}], manualAcks = true);
    check responder.attach(new Responder());
    check responder.'start();
    _ = check c->publish("listener-v/req", {payload: "q1".toBytes(),
        properties: {responseTopic: "listener-v/replies", correlationData: "c-1".toBytes()}});
    // The reply and the onComplete report travel independently, so print them in a fixed order.
    map<string> received = {};
    foreach int _ in 0 ..< 2 {
        record {|mqtt:Message value;|}? next = check events.next();
        if next is record {|mqtt:Message value;|} {
            mqtt:Message m = next.value;
            byte[]? correlation = m.properties?.correlationData;
            string suffix = correlation is byte[] ? " correlation=" + check string:fromBytes(correlation) : "";
            received[m.topic ?: ""] = check string:fromBytes(m.payload) + suffix;
        }
    }
    io:println(received["listener-v/replies"]); // @output re: q1 correlation=c-1
    io:println(received[EVENTS]); // @output onComplete listener-v/replies
    _ = check c->publish("listener-v/req", {payload: "q2".toBytes()});
    io:println(check nextEvent(events)); // @output respond failed: Response topic is not set

    // A service without onMessage gets the failure through onError.
    mqtt:Listener noOnMessage = check new (mqtt:DEFAULT_URL, "listener-v-no-on-message", {topic: "listener-v/none"});
    check noOnMessage.attach(new OnlyOnError());
    check noOnMessage.'start();
    _ = check c->publish("listener-v/none", {payload: "n".toBytes()});
    io:println(check nextEvent(events)); // @output onError method onMessage not found

    // A panic in onMessage is reported to onError.
    mqtt:Listener panicking = check new (mqtt:DEFAULT_URL, "listener-v-panicking", "listener-v/panic");
    check panicking.attach(new Panicking());
    check panicking.'start();
    _ = check c->publish("listener-v/panic", {payload: "p".toBytes()});
    io:println(check nextEvent(events)); // @output onError onMessage panicked on p

    // Caller->respond and Caller->complete report failures, here a connection already stopped.
    mqtt:Listener stopping = check new (mqtt:DEFAULT_URL, "listener-v-stopping", "listener-v/stop", manualAcks = true);
    check stopping.attach(new StoppingResponder(stopping));
    check stopping.'start();
    _ = check c->publish("listener-v/stop", {payload: "s".toBytes(), properties: {responseTopic: "listener-v/replies"}});
    io:println(check nextEvent(events)); // @output respond: Client is not connected, complete: Client is not connected

    // Starting without a service subscribes and drops incoming messages.
    mqtt:Listener unattached = check new (mqtt:DEFAULT_URL, "listener-v-unattached", "listener-v/unattached");
    check unattached.'start();
    _ = check c->publish("listener-v/unattached", {payload: "dropped".toBytes()});

    // detach unsubscribes but keeps the connection; attaching again re-subscribes.
    mqtt:Error? r = echo.detach(new Echo());
    io:println("detach: ", r is mqtt:Error ? r.message() : "ok"); // @output detach: ok
    r = echo.detach(new Echo());
    io:println("detach again: ", r is mqtt:Error ? r.message() : "ok"); // @output detach again: ok
    check echo.attach(new Echo());
    _ = check c->publish("listener-v/echo1", {payload: "c".toBytes()});
    io:println(check nextEvent(events)); // @output echo c qos=1
    r = echo.detach(new Echo());
    io:println("detach: ", r is mqtt:Error ? r.message() : "ok"); // @output detach: ok
    r = echo.gracefulStop();
    io:println("gracefulStop after detach: ", r is mqtt:Error ? r.message() : "ok"); // @output gracefulStop after detach: ok

    r = responder.gracefulStop();
    io:println("gracefulStop: ", r is mqtt:Error ? r.message() : "ok"); // @output gracefulStop: ok
    r = responder.gracefulStop();
    io:println("gracefulStop again: ", r is mqtt:Error ? r.message() : "ok"); // @output gracefulStop again: Client is closed
    r = noOnMessage.immediateStop();
    io:println("immediateStop: ", r is mqtt:Error ? r.message() : "ok"); // @output immediateStop: ok
    r = panicking.gracefulStop();
    io:println("gracefulStop panicking: ", r is mqtt:Error ? r.message() : "ok"); // @output gracefulStop panicking: ok
    r = unattached.immediateStop();
    io:println("immediateStop unattached: ", r is mqtt:Error ? r.message() : "ok"); // @output immediateStop unattached: ok

    check events.close();
    check c->disconnect();
    check c->close();
}
