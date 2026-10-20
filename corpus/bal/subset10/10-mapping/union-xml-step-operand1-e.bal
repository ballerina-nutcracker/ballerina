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

type A record {|
    xml v;
    int m;
|};

type B record {|
    xml v;
    string m;
|};

// Candidate trials suppress semantic errors, so the string operand of the step is not reported at the step
// itself: every candidate fails and the constructor reports that no inherent type applies.
public function main() {
    string s = "<a>1</a>";
    A|B m = {v: s/<a>, m: 1}; // @error no applicable inherent type for mapping constructor
    var _ = m;
}
