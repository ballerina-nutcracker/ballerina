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

// W is defined first, so X's defaults resolve while W's still wait. Only W
// fits X's default, through W's default names; selecting W resolves W's
// defaults, and the one that fails reports only its own diagnostic.
type W record {|
    int c = 1 / 0; // @error division by zero in W's default, reached when X's default selects W
    X? x = ();
|};

type X record {|
    int a = 1;
    W|V f = {};
|};

type V record {|
    int b;
|};

public function main() {
    X x = {};
    io:println(x);
}
