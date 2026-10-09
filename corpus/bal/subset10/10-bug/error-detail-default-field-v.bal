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

type Detail record {|
    int code = 42;
    string reason = "unknown";
|};

type DetailError error<Detail>;

type OpenDetail record {
    int count = 7;
};

type InlineError error<record {| int z = 3; |}>;

isolated function defaultCount() returns int {
    return 5;
}

type ComputedDetail record {|
    int count = defaultCount();
|};

type WideDetail record {| int x = 300; string label = "ok"; |};

type IntDetail record {| int x; string label; |};

type ByteDetail record {| byte x; string label; |};

type IntersectedError error<WideDetail & IntDetail>;

type NarrowedError error<WideDetail & ByteDetail>;

function inlineReturn() returns error<record {| int r = 10; |}> {
    return error("ret");
}

function inlineParam(error<record {| int p = 9; |}> e) {
    io:println(e);
}

public function main() {
    DetailError e1 = error("explicit", code = 1);
    io:println(e1); // @output error("explicit",code=1,reason="unknown")
    DetailError e2 = error("defaulted");
    io:println(e2); // @output error("defaulted",code=42,reason="unknown")
    DetailError e3 = error("all", code = 2, reason = "bad");
    io:println(e3); // @output error("all",code=2,reason="bad")
    error e4 = error DetailError("typeRef", reason = "ref");
    io:println(e4); // @output error DetailError ("typeRef",reason="ref",code=42)
    error<OpenDetail> e5 = error("open", extra = true);
    io:println(e5); // @output error("open",extra=true,count=7)
    InlineError e6 = error("inlineDef");
    io:println(e6); // @output error("inlineDef",z=3)
    error<record {| string tag = "local"; |}> e7 = error("inlineLocal");
    io:println(e7); // @output error("inlineLocal",tag="local")
    error<ComputedDetail> e8 = error("computed", count = 1);
    io:println(e8); // @output error("computed",count=1)
    error<ComputedDetail> e9 = error("computed");
    io:println(e9); // @output error("computed",count=5)
    io:println(inlineReturn()); // @output error("ret",r=10)
    inlineParam(error("param")); // @output error("param",p=9)
    IntersectedError e11 = error("intersected");
    io:println(e11); // @output error("intersected",x=300,label="ok")
    NarrowedError e12 = error("narrowed", x = 1);
    io:println(e12); // @output error("narrowed",x=1,label="ok")
    error cause = error("cause");
    DetailError e10 = error("withCause", cause, code = 5);
    io:println(e10); // @output error("withCause",error("cause"),code=5,reason="unknown")
}
