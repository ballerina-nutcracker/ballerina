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

function narrowFloat(float f) {
    if f is 1.5 {
        return;
    }
    int _ = f; // @error
}

function narrowDecimal(decimal d) {
    if d is 1.5d {
        return;
    }
    int _ = d; // @error
}

function narrowString(string s) {
    if s is "a"|"bc" {
        return;
    }
    int _ = s; // @error
}

function narrowNonChar(string s) {
    if s is string:Char {
        return;
    }
    int _ = s; // @error
}

function narrowChar(string:Char|"bc" s) {
    if s is "a" {
        return;
    }
    int _ = s; // @error
}

public function main() {
    2.0 f = 2.0;
    int _ = f; // @error
    2.0d d = 2.0d;
    int _ = d; // @error
}
