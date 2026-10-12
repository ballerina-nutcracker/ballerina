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

// Field types of a readonly class are intersected with readonly. stream & readonly is empty, so
// the field is reported and, as a consequence, the class type is empty too.
readonly class Numbers { // @error class definition is empty
    public stream<int, ()> values; // @error field type has no readonly values

    function init(stream<int, ()> values) {
        self.values = values;
    }
}

public function main() {
}
