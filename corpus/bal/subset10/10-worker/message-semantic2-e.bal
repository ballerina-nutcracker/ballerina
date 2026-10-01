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

class Counter {
    int count = 0;
}

function closedRecordExtraField() {
    worker a {
        1 -> function;
    }
    worker b {
        2 -> function;
    }
    record {|int a;|} _ = <- {a, b}; // @error the closed record has no field b
}

function anyValue() {
    worker a {
        any v = 1;
        v -> function; // @error any isn't Cloneable
    }
    any _ = <- a;
}

function nonCloneableSyncSend() {
    worker a {
        Counter _ = <- function;
    }
    _ = new Counter() ->> a; // @error an object isn't Cloneable
}
