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

# Documentation for File type
# + path - file path. Example: `C:\users\OddThinking\Documents\My Source\Widget\foo.src`
public class File {
    public string path = "";

    # Gets a access parameter value (`true` or `false`) for a given key. Please note that #foo will always be bigger than #bar.
    # Example:
    # `SymbolEnv pkgEnv = symbolEnter.packageEnvs.get(pkgNode.symbol);`
    # + accessMode - read or write mode
    # + return - success or not
    public function open(string accessMode) returns boolean {
        string _ = accessMode;
        return true;
    }
}

#
# + anUnion - value of param1
# + anInt - value of param2
# + rest - value of rest param
function insert(string|int|float anUnion, int anInt, string... rest) {
    string|int|float _ = anUnion;
    int _ = anInt;
    string[] _ = rest;
}

#
# + return - `string` value of the X will be returned if found, else an `error` will be returned
function getX() returns string|error {
    return "";
}

#
# + aTuple - a `Tuple` where the values should represent name, age and weight of a person, in order
# + anInt - index
function updatePeople([string, int, float] aTuple, int anInt) {
    [string, int, float] _ = aTuple;
    int _ = anInt;
}

#
# + return - a `Tuple` will be returned and would represent name, age and weight of a person, in order
function searchPeople() returns [string, int, float] {
    return ["", 1, 1.0];
}

# Test type `typeDef`
# Test service `helloWorld`
# Test variable `testVar`
# Test var `testVar`
# Test function `add`
# Test parameter `x`
# Test const `constant`
# Test annotation `annot`
# + x - one thing to be added
# + y - another thing to be added
# + return - the sum of them
function add(int x, int y) returns int {
    return x + y;
}

# + 'example - The error struct to be logged
function foo(error 'example) {
    if 'example is error {
    }
}

# + 'foo_\u{2345}\u{0376} - The error struct to be logged with unicode name
function foo_unicode(error 'foo_\u{2345}\u{0376}) {
    if 'foo_\u{2345}\u{0376} is error {
    }
}
