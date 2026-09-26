// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// An index extension accepts only int. Each later extension operates on the XML
// sequence produced by the preceding extension.
function testXmlNegativeIndexedAndFilterStepExtend() {
    int j = 0;
    string s = "a";
    xml x1 = xml `<item><name>T-shirt</name><price>19.99</price></item>`;
    _ = x1/*[j].<name>;
    _ = x1/*[s].<name>; // @error
    xml x2 = x1;
    _ = x2/*.<s>;
    _ = x2/*.<s>[j];
}

// Method-call extensions are mapped over the selected XML items. The method
// must therefore be a lang.xml method with matching arguments and an XML result;
// filter callbacks must return boolean, and map callbacks are checked against
// the selected item type (xml:Element for the named steps below).
function testXmlMethodCallNegativeStepExtend() returns error? {
    int k = 0;
    int r = 0;
    string s = "s";

    xml x1 = xml `<item><name>T-shirt</name><price>19.99</price></item>`;

    _ = x1/*.get("0"); // @error
    _ = x1/*.get(0, 2); // @error
    _ = x1/*.get(s); // @error

    _ = x1/*.foo(); // @error
    _ = x1/*.length(); // @error
    _ = x1/*.slice(1, "5"); // @error

    _ = x1/*.slice(0).length(); // @error
    _ = x1/*.elementChildren().get(0).getChildren();


    _ = x1/<item>.get(r);
    _ = x1/<item>.get(0).getChildren();
    _ = x1/<item>.filter(x => x); // @error

    _ = x1/<item>.map(y => xml:createProcessingInstruction(y.getTarget(), "sort")); // @error
    _ = x1/<item>.forEach(function (xml y) {_ = y; k = k + 1;}); // @error

    _ = x1/**/<item>.get(r);
    _ = x1/**/<item>.get(0).getChildren();
    _ = x1/**/<item>.filter(x => x); // @error
    _ = x1/**/<item>.filter(function (xml y) {_ = y; k = k + 1;}); // @error
}
