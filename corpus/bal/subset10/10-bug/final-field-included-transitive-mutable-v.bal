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

class Base {
    final int a = 1;
}

type View object {
    *Base;
};

class Middle {
    *View;

    function init() {
        self.a = 2;
    }
}

class Leaf {
    *Middle;

    function init() {
        self.a = 3;
        self.a = 4;
        self.a += 1;
    }

    function update() {
        self.a = 6;
    }
}

class ExplicitFinal {
    *Base;
    final int a = 8;
}

function write(object { int a; } value) {
    value.a += 1;
}

public function main() {
    var value = new Leaf();
    value.update();
    value.a += 1;
    io:println(value.a); // @output 7
    Base original = new;
    error? sourceResult = trap write(original);
    io:println(sourceResult is error); // @output true
    io:println(original.a); // @output 1
    ExplicitFinal explicitFinal = new;
    error? explicitResult = trap write(explicitFinal);
    io:println(explicitResult is error); // @output true
    io:println(explicitFinal.a); // @output 8
}
