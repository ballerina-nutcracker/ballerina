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

type Point object {
    public int x;
};

type Items object {
    public int[] items;
};

readonly class ReadonlyPoint {
    public int x = 7;
}

readonly class ReadonlyList {
    *Items;
    public int[] values = [1, 2];

    function init(int[] & readonly items) {
        self.items = items;
    }
}

readonly client class ReadonlyClient {
    remote function ping() returns int {
        return 1;
    }

    resource function get count() returns int {
        return 2;
    }
}

isolated readonly class IsolatedReadonlyPoint {
    public int x = 4;

    isolated function getX() returns int {
        return self.x;
    }
}

type ReadonlyGetter readonly & object {
    public int x;
    isolated function getX() returns int;
};

class MutablePoint {
    public int x = 1;
}

public function main() {
    readonly & Point point = new ReadonlyPoint();
    io:println(point.x); // @output 7

    any value = new ReadonlyPoint();
    io:println(value is readonly); // @output true
    io:println(value is readonly & Point); // @output true

    ReadonlyPoint readonlyPoint = new;
    readonly readonlyValue = readonlyPoint;
    io:println(readonlyValue is ReadonlyPoint); // @output true

    ReadonlyList list = new ([3]);
    int[] & readonly values = list.values;
    io:println(values); // @output [1,2]
    any listValue = list.values;
    io:println(listValue is readonly); // @output true
    any items = list.items;
    io:println(items is readonly); // @output true

    ReadonlyClient readonlyClient = new;
    any clientValue = readonlyClient;
    io:println(clientValue is readonly); // @output true
    int pinged = readonlyClient->ping();
    io:println(pinged); // @output 1

    ReadonlyGetter getter = new IsolatedReadonlyPoint();
    io:println(getter.getX()); // @output 4

    any mutableValue = new MutablePoint();
    io:println(mutableValue is readonly); // @output false
}
