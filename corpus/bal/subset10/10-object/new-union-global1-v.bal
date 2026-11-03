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

class IntBox {
    int value;

    function init(int value, int marker) {
        self.value = value + marker;
    }
}

class StringBox {
    string value;

    function init(string value, int marker) {
        self.value = value;
        var _ = marker;
    }
}

IntBox|StringBox box = new (<Count>(later + 1), 1);

int later = 4;

type Count int;

public function main() {
    if box is IntBox {
        io:println(box.value); // @output 6
    }
}
