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

isolated class IfBranch {
    private int[] arr;

    isolated function init(boolean b, int[] a) {
        if b {
            self.arr = a; // @error
        } else {
            self.arr = [];
        }
    }
}

isolated class ElseBranch {
    private int[] arr;

    isolated function init(boolean b, int[] a) {
        if b {
            self.arr = [];
        } else {
            self.arr = a; // @error
        }
    }
}

isolated class WhileLoop {
    private int[] arr = [];

    isolated function init(int[] a) {
        int i = 0;
        while i < 2 {
            self.arr = a; // @error
            i += 1;
        }
    }
}

isolated class ForeachLoop {
    private int[] arr = [];

    isolated function init(int[][] a) {
        foreach int[] x in a {
            self.arr = x; // @error
        }
    }
}

isolated class MatchClause {
    private int[] arr = [];

    isolated function init(int k, int[] a) {
        match k {
            1 => {
                self.arr = a; // @error
            }
        }
    }
}

isolated class NestedBlock {
    private int[] arr = [];

    isolated function init(int[] a) {
        {
            self.arr = a; // @error
        }
    }
}

public function main() {
    _ = new IfBranch(true, [1]);
    _ = new ElseBranch(false, [1]);
    _ = new WhileLoop([1]);
    _ = new ForeachLoop([[1]]);
    _ = new MatchClause(1, [1]);
    _ = new NestedBlock([1]);
}
