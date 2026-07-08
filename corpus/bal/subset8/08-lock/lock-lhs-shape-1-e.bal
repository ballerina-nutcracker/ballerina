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

isolated int counter = 0;

function bumpAt(int[] arr) {
    lock {
        // Strictly according to spec this shouldn't be allowed but jBallerina allows this I assume this because counter
        // don't have storage identity. We partially implement this as well.
        arr[0] = counter; // @error LHS is index access on an outside-lock array
    }
}

public function main() {
    bumpAt([0, 0]);
}
