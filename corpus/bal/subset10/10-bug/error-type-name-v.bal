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

type E error;

type MyDetail record {
    int code;
};

type MyError error<MyDetail>;

type AliasError MyError;

type DistinctError distinct error;

type OtherDistinctError distinct error;

type AliasOfDistinctError DistinctError;

type BothDistinctError DistinctError & OtherDistinctError;

type ReadonlyError readonly & MyError;

public function main() {
    E e1 = error("x");
    E e2 = error E("x");
    io:println(e1); // @output error("x")
    io:println(e2); // @output error("x")

    io:println(error MyError("m", code = 5)); // @output error("m",code=5)
    io:println(error AliasError("a", code = 1)); // @output error("a",code=1)
    io:println(error ReadonlyError("r", code = 2)); // @output error("r",code=2)

    (any|error)[] list = [e2, error MyError("m", code = 5)];
    io:println(list.toBalString()); // @output [error("x"),error("m",code=5)]

    // Types with type-ids render with the referenced type name. Spec 5.9.5
    // renders the type-ids instead, which is not implemented yet.
    io:println(error DistinctError("d")); // @output error DistinctError ("d")
    io:println(error AliasOfDistinctError("ad")); // @output error AliasOfDistinctError ("ad")
    io:println(error BothDistinctError("b")); // @output error BothDistinctError ("b")
}
