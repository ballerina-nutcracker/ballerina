// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// parseJSON decodes text the way jBallerina's getJson does: object keys keep their
// source order, a negative zero becomes float -0.0, and any other number with a
// fraction or exponent becomes a decimal.
func (t *mimeTypes) parseJSON(text string) (values.BalValue, error) {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	v, err := t.decodeJSONValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing characters after the JSON value")
	}
	return v, nil
}

func (t *mimeTypes) decodeJSONValue(dec *json.Decoder) (values.BalValue, error) {
	tok, err := dec.Token()
	if errors.Is(err, io.EOF) {
		return nil, io.ErrUnexpectedEOF
	}
	if err != nil {
		return nil, err
	}
	switch tok := tok.(type) {
	case json.Delim:
		if tok == '[' {
			return t.decodeJSONArray(dec)
		}
		return t.decodeJSONObject(dec)
	case json.Number:
		return jsonNumberToBal(tok)
	default:
		return tok, nil
	}
}

func (t *mimeTypes) decodeJSONArray(dec *json.Decoder) (values.BalValue, error) {
	items := make([]values.BalValue, 0)
	for dec.More() {
		item, err := t.decodeJSONValue(dec)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return values.NewList(t.jsonListTy, t.jsonListAtom, false, nil, 0, items), nil
}

func (t *mimeTypes) decodeJSONObject(dec *json.Decoder) (values.BalValue, error) {
	var entries []values.MapEntry
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		value, err := t.decodeJSONValue(dec)
		if err != nil {
			return nil, err
		}
		entries = append(entries, values.MapEntry{Key: keyTok.(string), Value: value})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return values.NewMap(t.jsonMapTy, t.jsonMapAtom, false, entries), nil
}

func jsonNumberToBal(n json.Number) (values.BalValue, error) {
	s := n.String()
	if isNegativeZero(s) {
		return math.Copysign(0, -1), nil
	}
	if !strings.ContainsAny(s, ".eE") {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, nil
		}
	}
	d, err := decimal.FromLiteral(s)
	if err != nil {
		return nil, fmt.Errorf("number out of range: %s", s)
	}
	return d, nil
}

func isNegativeZero(s string) bool {
	if !strings.HasPrefix(s, "-") {
		return false
	}
	f, err := strconv.ParseFloat(s, 64)
	return err == nil && f == 0
}
