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

client class Client {
    remote function scale(int i) returns int {
        return i * 2;
    }
}

class Counter {
    int n = 3;

    function next() returns int {
        return self.n + 1;
    }
}

function one() returns int {
    return 1;
}

function times(int i) returns int {
    return i * 10;
}

public function main() {
    future<int> braced = <future<int>>(start one());
    int bracedValue = checkpanic wait braced;
    io:println(bracedValue); // @output 1

    future<int> bare = <future<int>> start times(2);
    int bareValue = checkpanic wait bare;
    io:println(bareValue); // @output 20

    any widened = <any> start times(3);
    if widened is future<int> {
        int widenedValue = checkpanic wait widened;
        io:println(widenedValue); // @output 30
    }

    Counter c = new;
    future<int> method = <future<int>> start c.next();
    int methodValue = checkpanic wait method;
    io:println(methodValue); // @output 4

    Client cl = new;
    int remoteValue = <int> cl->scale(5);
    io:println(remoteValue); // @output 10

    future<int> remoteFuture = <future<int>> start cl->scale(9);
    int remoteFutureValue = checkpanic wait remoteFuture;
    io:println(remoteFutureValue); // @output 18

    any nested = <any> <future<int>> start times(6);
    if nested is future<int> {
        int nestedValue = checkpanic wait nested;
        io:println(nestedValue); // @output 60
    }

    int base = 1;
    var startLater = function(int k) returns future<int> {
        return <future<int>> start times(k + base);
    };
    int closureValue = checkpanic wait startLater(6);
    io:println(closureValue); // @output 70
}
