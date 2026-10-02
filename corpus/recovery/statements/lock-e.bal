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

int[] outside = [0];

isolated class LockRecovery {
    int[] state = [];

    function update() {
        lock {
            self.state = [];
            self.missing = 1; // @error abandoned assignment
            self.missing += 1; // @error abandoned compound assignment
            if missingCondition { // @error abandoned compound statement
                self.anotherMissing = 1; // @error symbol resolution still visits body
                outside[0] = 2;
            }
            int local = 0;
            local = 1;
            outside[0] = 1; // @error independent lock transfer check
        }
        int later = "wrong"; // @error independent semantic analysis
    }
}
