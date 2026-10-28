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

class FinalView {
    final int a = 1;
}

class Mutable {
    int a = 1;
}

type Alias FinalView;
type AliasChain Alias;

function write(AliasChain value) {
    value.a += 2;
}

public function main() {
    Mutable value = new;
    write(value);
    io:println(value.a); // @output 3
    AliasChain finalValue = new FinalView();
    error? result = trap write(finalValue);
    io:println(result is error); // @output true
    io:println(finalValue.a); // @output 1
}
