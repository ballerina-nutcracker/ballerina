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
	"ballerina/decimal"
)

type decimalOps struct {
	CommonOps
}

var _ BasicTypeOps = &decimalOps{}

func newDecimalOps() decimalOps {
	this := decimalOps{}
	return this
}

func (d *decimalOps) Union(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	var values []enumerableType[decimal.Decimal]
	var v1 enumerableSubtype[decimal.Decimal] = new(t1.(decimalSubtype))
	var v2 enumerableSubtype[decimal.Decimal] = new(t2.(decimalSubtype))
	allowed := enumerableSubtypeUnion(v1, v2, &values)
	return createDecimalSubtype(allowed, values)
}

func (d *decimalOps) Intersect(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	var values []enumerableType[decimal.Decimal]
	var v1 enumerableSubtype[decimal.Decimal] = new(t1.(decimalSubtype))
	var v2 enumerableSubtype[decimal.Decimal] = new(t2.(decimalSubtype))
	allowed := enumerableSubtypeIntersect(v1, v2, &values)
	return createDecimalSubtype(allowed, values)
}

func (d *decimalOps) Diff(t1 SubtypeData, t2 SubtypeData) SubtypeData {
	return d.Intersect(t1, d.complement(t2))
}

func (d *decimalOps) complement(t SubtypeData) SubtypeData {
	s := t.(decimalSubtype)
	return createDecimalSubtype((!s.allowed), s.Values())
}

func (d *decimalOps) IsEmpty(tc Context, t SubtypeData) bool {
	return notIsEmpty(tc, t)
}
