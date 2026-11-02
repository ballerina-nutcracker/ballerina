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
type IsoObj isolated object {
};

isolated class Iso {
}

class Plain {
}

isolated class Holder {
    IsoObj a = new Iso(); // @error isolated object type but not final
    public Iso b = new; // @error public isolated class type but not final
    final IsoObj|Plain c = new Plain(); // @error union with a non-isolated object type
    final any d = 1; // @error any is not a subtype of readonly
    private IsoObj e = new Iso();
}

public function main() {
    Holder _ = new;
}
