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

type Rec record {|
    int a;
|};

class Plain {
    int x = 0;
}

isolated class Holder {
    final anydata data = 1; // @error anydata is not a subtype of readonly
    final map<int> counts; // @error mutable map type, even though final and assigned in init
    final Rec rec = {a: 1}; // @error mutable record type
    public final int[] nums = []; // @error public final field with mutable list type
    final Plain plain = new; // @error non-isolated object type is not readonly
    final readonly & Rec ok = {a: 2};
    final string name;
    final int|string? tag = ();
    final readonly val = 7;
    final error err = error("boom");

    function init() {
        self.counts = {};
        self.name = "n";
    }
}

public function main() {
    Holder _ = new;
}
