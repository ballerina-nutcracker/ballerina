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

package semtypes

import (
	"ballerina-lang-go/decimal"
)

type enumerableDecimal struct {
	value decimal.Decimal
}

var _ enumerableType[decimal.Decimal] = &enumerableDecimal{}

func (e *enumerableDecimal) Value() decimal.Decimal {
	return e.value
}

func (t1 *enumerableDecimal) Compare(t2 enumerableType[decimal.Decimal]) int {
	f1 := t1.Value()
	f2 := t2.Value()
	return f1.Cmp(&f2)
}

func enumerableDecimalFrom(d decimal.Decimal) enumerableDecimal {
	return enumerableDecimal{value: d}
}
