// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package corpus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

// spliceTopLevelJSONField replaces the value bound to key at raw's top level
// with newValue, leaving every other byte of raw -- key order, spacing,
// every other field -- exactly as written. Unlike a decode/re-encode round
// trip (which reformats the whole document and may reorder map keys), this
// touches only the byte span of key's own value.
func spliceTopLevelJSONField(raw []byte, key string, newValue []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	open, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := open.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("spliceTopLevelJSONField: not a JSON object")
	}
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keyName, _ := keyToken.(string)
		afterKey := dec.InputOffset()
		if keyName != key {
			var discard json.RawMessage
			if err := dec.Decode(&discard); err != nil {
				return nil, err
			}
			continue
		}
		valueStart, err := skipToValue(raw, afterKey)
		if err != nil {
			return nil, err
		}
		var discard json.RawMessage
		if err := dec.Decode(&discard); err != nil {
			return nil, err
		}
		valueEnd := dec.InputOffset()
		spliced := make([]byte, 0, len(raw)-int(valueEnd-int64(valueStart))+len(newValue))
		spliced = append(spliced, raw[:valueStart]...)
		spliced = append(spliced, newValue...)
		spliced = append(spliced, raw[valueEnd:]...)
		return spliced, nil
	}
	return nil, fmt.Errorf("spliceTopLevelJSONField: key %q not found", key)
}

// skipToValue returns the offset of the first non-whitespace byte after the
// colon following a key, given the byte offset right after the key token.
func skipToValue(raw []byte, afterKey int64) (int, error) {
	colon := bytes.IndexByte(raw[afterKey:], ':')
	if colon < 0 {
		return 0, fmt.Errorf("skipToValue: missing colon")
	}
	start := int(afterKey) + colon + 1
	for start < len(raw) && isJSONSpace(raw[start]) {
		start++
	}
	return start, nil
}

func isJSONSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func TestSpliceTopLevelJSONFieldReplacesOnlyTargetValue(t *testing.T) {
	raw := []byte(`{
  "position": {"line": 1, "character": 2},
  "source": "type_def/source/source1.bal",
  "items": [
    {"label": "old"}
  ]
}
`)
	got, err := spliceTopLevelJSONField(raw, "items", []byte(`[{"label":"new"}]`))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte(`{
  "position": {"line": 1, "character": 2},
  "source": "type_def/source/source1.bal",
  "items": [{"label":"new"}]
}
`)
	if !bytes.Equal(got, want) {
		t.Fatalf("spliceTopLevelJSONField() =\n%s\nwant\n%s", got, want)
	}
}

func TestSpliceTopLevelJSONFieldPreservesUnrelatedBytes(t *testing.T) {
	raw := []byte("{\"a\":1,   \"items\"\n:\n   [1,2,3],\"z\":\"tail\"}")
	got, err := spliceTopLevelJSONField(raw, "items", []byte(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("{\"a\":1,   \"items\"\n:\n   [],\"z\":\"tail\"}")
	if !bytes.Equal(got, want) {
		t.Fatalf("spliceTopLevelJSONField() = %q, want %q", got, want)
	}
}

func TestSpliceTopLevelJSONFieldMissingKey(t *testing.T) {
	if _, err := spliceTopLevelJSONField([]byte(`{"a":1}`), "items", []byte(`[]`)); err == nil {
		t.Fatal("expected an error for a missing key")
	}
}

func TestSpliceTopLevelJSONFieldNotAnObject(t *testing.T) {
	if _, err := spliceTopLevelJSONField([]byte(`[1,2,3]`), "items", []byte(`[]`)); err == nil {
		t.Fatal("expected an error for a non-object document")
	}
}
