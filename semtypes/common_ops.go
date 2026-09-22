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

type commonOps = commonBasicTypeOps

type commonOpsBase struct {
	commonOpsMethods
}

type commonOpsMethods struct {
	Self commonOps
}

func (m *commonOpsMethods) Union(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddUnion(t1.(bdd), t2.(bdd))
}

func (m *commonOpsMethods) Intersect(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddIntersect(t1.(bdd), t2.(bdd))
}

func (m *commonOpsMethods) Diff(t1 subtypeData, t2 subtypeData) subtypeData {
	return bddDiff(t1.(bdd), t2.(bdd))
}

func (m *commonOpsMethods) complement(t subtypeData) subtypeData {
	return bddComplement(t.(bdd))
}
