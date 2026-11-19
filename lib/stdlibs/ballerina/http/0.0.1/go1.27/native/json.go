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

package native

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// toJSONBytes serialises an anydata value as JSON the way jBallerina's toJson does: mapping
// members keep their order, xml is written as its string form, and strings are not
// HTML-escaped. A value with no JSON form, such as a NaN or infinite float, is an error.
func toJSONBytes(v values.BalValue) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeJSON(buf *bytes.Buffer, v values.BalValue) error {
	switch t := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		buf.WriteString(strconv.FormatBool(t))
	case int64:
		buf.WriteString(strconv.FormatInt(t, 10))
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return fmt.Errorf("unsupported float value: %v", t)
		}
		b, _ := json.Marshal(t)
		buf.Write(b)
	case *decimal.Decimal:
		buf.WriteString(t.String())
	case string:
		writeJSONString(buf, t)
	case values.XMLValue:
		writeJSONString(buf, t.XMLString())
	case *values.Map:
		buf.WriteByte('{')
		for i, key := range t.Keys() {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJSONString(buf, key)
			buf.WriteByte(':')
			member, _ := t.Get(key)
			if err := writeJSON(buf, member); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case *values.List:
		buf.WriteByte('[')
		for i := range t.Len() {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeJSON(buf, t.Get(i)); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		return fmt.Errorf("unsupported value of type %T", v)
	}
	return nil
}

func writeJSONString(buf *bytes.Buffer, s string) {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	buf.Truncate(buf.Len() - 1)
}
