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

function inBlock() {
    worker a {
        1 -> function;
    }
    {
        _ = wait a; // @error wait on a worker is not supported here when the workers exchange messages
    }
    int _ = <- a;
}

function inIfElse(boolean flag) {
    worker a {
        int _ = <- function;
    }
    if flag {
        _ = wait a; // @error wait on a worker is not supported here when the workers exchange messages
    } else {
    }
    1 -> a;
}

function afterIfWithoutElse() returns int {
    worker a {
        1 -> function;
    }
    worker b returns int {
        return 2;
    }
    int x = <- a;
    if x < 0 {
        return x;
    }
    int y = wait b; // @error the statements after an if without else form a nested block
    return x + y;
}
