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

type Tag record {|
    string value;
|};

// Repeated annotation — annotation type is Tag[] (map<Cloneable>[]).
annotation Tag[] tags on type;

@tags {value: "first"}
@tags {value: "second"}
@tags {value: "third"}
type Tagged int;

public function main() {
    Tag[]? tagValues = Tagged.@tags;
    if tagValues is Tag[] {
        foreach var t in tagValues {
            io:println(t.value); // @output first
                                 // @output second
                                 // @output third
        }
    }
}
