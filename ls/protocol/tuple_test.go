// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protocol

import (
	"encoding/json"
	"testing"
)

func TestSignatureLabelTuple(t *testing.T) {
	want := `[4,13]`
	tuple := NewOrParameterInformationLabelVariant1(TupleParameterInformationLabelItem1{Item0: 4, Item1: 13})
	got, err := json.Marshal(tuple)
	if err != nil || string(got) != want {
		t.Fatalf("encode: %s %v", got, err)
	}
	var decoded OrParameterInformationLabel
	if err := json.Unmarshal([]byte(want), &decoded); err != nil {
		t.Fatal(err)
	}
	value, ok := decoded.Variant1()
	if !ok || value.Item0 != 4 || value.Item1 != 13 {
		t.Fatalf("decode: %+v", decoded)
	}
	for _, invalid := range []string{`[]`, `[1]`, `[1,2,3]`, `null`, `{"Item0":1,"Item1":2}`, `[-1,2]`, `[1,"2"]`} {
		var value TupleParameterInformationLabelItem1
		if json.Unmarshal([]byte(invalid), &value) == nil {
			t.Errorf("accepted %s", invalid)
		}
	}
}
