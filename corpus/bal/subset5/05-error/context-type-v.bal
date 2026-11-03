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
    ErrA? errA;
|};

type ErrA error<Detail>;

public function main() {
    ErrA err0 = error ErrA("whoops", errA = ());
    io:println(err0); // @output error("whoops",errA=null)
    ErrA error1 = error("Whoops", errA = ());
    io:println(error1); // @output error("Whoops",errA=null)
    ErrA error2 = error("Whoops", errA = error1);
    io:println(error2); // @output error("Whoops",errA=error("Whoops",errA=null))
}
