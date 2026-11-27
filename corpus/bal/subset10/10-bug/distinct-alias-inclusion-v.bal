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

type Shape distinct object {
    public function area() returns int;
};

// A type defined as a distinct type, without being distinct itself, denotes
// the same type, so including it includes the distinct type.
type ShapeAlias Shape;

class Square {
    *ShapeAlias;

    public function area() returns int => 4;
}

class Lookalike {
    public function area() returns int => 4;
}

public function main() {
    Shape s = new Square();
    io:println(s.area()); // @output 4
    any a = new Square();
    io:println(a is Shape); // @output true
    io:println(a is ShapeAlias); // @output true
    any b = new Lookalike();
    io:println(b is Shape); // @output false
}
