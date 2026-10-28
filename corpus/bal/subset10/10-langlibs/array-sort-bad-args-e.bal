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

type Person record {|
    string name;
|};

public function main() {
    Person[] people = [{name: "a"}];
    _ = people.sort(); // @error sort key must have an ordered type
    (int|string)[] mixed = [1, "a"];
    _ = mixed.sort(); // @error sort key must have an ordered type
    int[] values = [1];
    _ = values.sort(key = isolated function(int v) returns Person => {name: v.toString()}); // @error sort key must have an ordered type
    _ = values.sort("up"); // @error direction must be an array:SortDirection
}
