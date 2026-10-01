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

// @productions match-stmt match-guard int-literal string-literal any type-union function-call-expr
public function main() {
    unionGuard(1, true);
    callGuard(1);
    laterClause(1, true);
    nested(1, 2);
    anyGuard(1, true);
}

function unionGuard(int v, boolean|int b) {
    match v {
        _ if b => { // @error
        }
    }
}

function callGuard(int v) {
    match v {
        _ if count() => { // @error
        }
    }
}

function count() returns int {
    return 1;
}

function laterClause(int v, boolean ok) {
    match v {
        1 if ok => {
        }
        2 if v => { // @error
        }
        _ if "x" => { // @error
        }
    }
}

function nested(int v, int w) {
    match v {
        _ => {
            match w {
                _ if w => { // @error
                }
            }
        }
    }
}

function anyGuard(int v, any a) {
    match v {
        _ if a => { // @error
        }
    }
}
