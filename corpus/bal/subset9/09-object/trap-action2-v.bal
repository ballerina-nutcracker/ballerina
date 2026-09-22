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

int calls = 0;

function failInt(string message) returns int {
    panic error(message);
}

function failValues() returns int[] {
    panic error("spread panic");
}

function failClient() returns Client {
    panic error("receiver panic");
}

client class Client {
    remote function value(int value = failInt("remote default panic")) returns int {
        calls += 1;
        return value;
    }

    resource function get item/[int id](int value = failInt("resource default panic")) returns int {
        calls += 1;
        return id + value;
    }

    remote function values(int[] values) returns int {
        calls += 1;
        return values.length();
    }

    resource function get values(int[] values) returns int {
        calls += 1;
        return values.length();
    }
}

function message(int|error result) returns string {
    return result is error ? result.message() : "not trapped";
}

public function main() {
    Client c = new;
    var remoteReceiver = trap (failClient())->value(1);
    io:println(message(remoteReceiver)); // @output receiver panic
    var resourceReceiver = trap (failClient())->/item/[1].get(1);
    io:println(message(resourceReceiver)); // @output receiver panic

    var path = trap c->/item/[failInt("path panic")].get(1);
    io:println(message(path)); // @output path panic
    var remoteArgument = trap c->value(failInt("remote argument panic"));
    io:println(message(remoteArgument)); // @output remote argument panic
    var resourceArgument = trap c->/item/[1].get(failInt("resource argument panic"));
    io:println(message(resourceArgument)); // @output resource argument panic

    var remoteDefault = trap c->value();
    io:println(message(remoteDefault)); // @output remote default panic
    var resourceDefault = trap c->/item/[1];
    io:println(message(resourceDefault)); // @output resource default panic

    boolean useSpread = true;
    var remoteSpread = trap c->values(useSpread ? [0, ...failValues()] : []);
    io:println(message(remoteSpread)); // @output spread panic
    var resourceSpread = trap c->/values.get(useSpread ? [0, ...failValues()] : []);
    io:println(message(resourceSpread)); // @output spread panic
    io:println(calls); // @output 0

    var remoteValue = trap c->value(5);
    io:println(remoteValue); // @output 5
    var resourceValue = trap c->/item/[2].get(3);
    io:println(resourceValue); // @output 5
    io:println(calls); // @output 2
}
