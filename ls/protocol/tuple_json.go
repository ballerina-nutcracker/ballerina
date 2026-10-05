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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package protocol

import (
	"encoding/json"
	"fmt"
)

func (t TupleParameterInformationLabelItem1) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Item0, t.Item1})
}

func (t *TupleParameterInformationLabelItem1) UnmarshalJSON(data []byte) error {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	if len(items) != 2 {
		return fmt.Errorf("tuple requires 2 elements")
	}
	var value TupleParameterInformationLabelItem1
	if string(items[0]) == "null" {
		return fmt.Errorf("tuple element cannot be null")
	}
	if err := json.Unmarshal(items[0], &value.Item0); err != nil {
		return err
	}
	if string(items[1]) == "null" {
		return fmt.Errorf("tuple element cannot be null")
	}
	if err := json.Unmarshal(items[1], &value.Item1); err != nil {
		return err
	}
	*t = value
	return nil
}
