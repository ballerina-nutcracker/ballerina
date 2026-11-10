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

type OpenDetail record {
    int count?;
};

type OpenErr error<OpenDetail>;

type RoDetail readonly & record {
    int code;
};

type RoErr error<RoDetail>;

type RestDetail record {|
    int code;
    string...;
|};

type DistinctErr distinct error<OpenDetail>;

function inferred() {
    error<OpenDetail> e = error("open", extra = true); // @error
    io:println(e);
}

function typeRef() {
    OpenErr e = error OpenErr("open", count = 1, extra = 2); // @error
    io:println(e);
}

function readonlyDetail() {
    error<RoDetail> e = error("ro", code = 1, other = "x"); // @error
    io:println(e);
}

function typeRefReadonlyDetail() {
    RoErr e = error RoErr("ro", code = 1, other = "x"); // @error
    io:println(e);
}

function restDescriptor() {
    error<RestDetail> e = error("rest", code = 1, other = "x"); // @error
    io:println(e);
}

function unionContext() {
    error<OpenDetail>? e = error("opt", extra = true); // @error
    io:println(e);
}

function distinctTypeRef() {
    DistinctErr e = error DistinctErr("distinct", extra = 1); // @error
    io:println(e);
}

public function main() {
    inferred();
    typeRef();
    readonlyDetail();
    typeRefReadonlyDetail();
    restDescriptor();
    unionContext();
    distinctTypeRef();
}
