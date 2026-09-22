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

type floatOps struct {
	commonOps
}

var _ basicTypeOps = &floatOps{}

func newFloatOps() floatOps {
	this := floatOps{}
	return this
}

func (f *floatOps) Union(t1 subtypeData, t2 subtypeData) subtypeData {
	var values []enumerableType[float64]
	var v1 enumerableSubtype[float64] = new(t1.(floatSubtype))
	var v2 enumerableSubtype[float64] = new(t2.(floatSubtype))
	allowed := enumerableSubtypeUnion(v1, v2, &values)
	return createFloatSubtype(allowed, values)
}

func (f *floatOps) Intersect(t1 subtypeData, t2 subtypeData) subtypeData {
	var values []enumerableType[float64]
	var v1 enumerableSubtype[float64] = new(t1.(floatSubtype))
	var v2 enumerableSubtype[float64] = new(t2.(floatSubtype))
	allowed := enumerableSubtypeIntersect(v1, v2, &values)
	return createFloatSubtype(allowed, values)
}

func (f *floatOps) Diff(t1 subtypeData, t2 subtypeData) subtypeData {
	return f.Intersect(t1, f.complement(t2))
}

func (f *floatOps) complement(t subtypeData) subtypeData {
	s := t.(floatSubtype)
	return createFloatSubtype((!s.allowed), s.Values())
}

func (f *floatOps) IsEmpty(cx Context, t subtypeData) bool {
	return notIsEmpty(cx, t)
}
