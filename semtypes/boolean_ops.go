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

type booleanOps struct{}

var _ basicTypeOps = &booleanOps{}

func (b *booleanOps) Union(d1 subtypeData, d2 subtypeData) subtypeData {
	v1 := d1.(booleanSubtype)
	v2 := d2.(booleanSubtype)
	if v1.Value == v2.Value {
		return v1
	} else {
		return createAll()
	}
}

func (b *booleanOps) Intersect(d1 subtypeData, d2 subtypeData) subtypeData {
	v1 := d1.(booleanSubtype)
	v2 := d2.(booleanSubtype)
	if v1.Value == v2.Value {
		return v1
	} else {
		return createNothing()
	}
}

func (b *booleanOps) Diff(d1 subtypeData, d2 subtypeData) subtypeData {
	v1 := d1.(booleanSubtype)
	v2 := d2.(booleanSubtype)
	if v1.Value == v2.Value {
		return createNothing()
	} else {
		return v1
	}
}

func (b *booleanOps) complement(d subtypeData) subtypeData {
	v := d.(booleanSubtype)
	t := booleanSubtypeFrom(!v.Value)
	return t
}

func (b *booleanOps) IsEmpty(cx Context, t subtypeData) bool {
	return notIsEmpty(cx, t)
}
