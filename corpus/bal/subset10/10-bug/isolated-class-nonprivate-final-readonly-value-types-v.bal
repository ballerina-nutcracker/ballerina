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

type IsoObj isolated object {
    isolated function id() returns int;
};

isolated class Iso {
    *IsoObj;

    isolated function id() returns int {
        return 7;
    }
}

isolated function double(int a) returns int => a * 2;

isolated class Holder {
    final isolated function (int) returns int f = double;
    final typedesc<int> td = int;
    public final IsoObj? maybe = new Iso();
    final IsoObj? none = ();
}

public function main() {
    Holder h = new;
    var f = h.f;
    io:println(f(4)); // @output 8
    io:println(h.td is typedesc<int>); // @output true
    IsoObj? maybe = h.maybe;
    if maybe is IsoObj {
        io:println(maybe.id()); // @output 7
    }
    io:println(h.none is ()); // @output true
}
