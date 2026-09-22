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

type First object {
    public function value(int input = 1) returns int;
};

type Second object {
    public function value(int input = 2) returns int;
};

type Third object {
    public function other(int input = 2) returns int;
};

function unionReceiver(First|Second receiver) returns int {
    return receiver.value(input = 3); // @error named args on union receiver
}

function intersectReceiver(First & Third receiver) returns int {
    return receiver.value(input = 4); // @error ambiguous declaring atom
}

function functionValueReceiver(function receiver) {
    receiver(input = 5); // @error named args without an untyped function signature
}
