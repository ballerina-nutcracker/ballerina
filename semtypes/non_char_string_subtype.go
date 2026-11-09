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

type nonCharStringSubtype struct {
	allowed bool
	values  []enumerableType[string]
}

var _ enumerableSubtype[string] = &nonCharStringSubtype{}

func newNonCharStringSubtypeFromBoolEnumerableStrings(allowed bool, values []enumerableType[string]) nonCharStringSubtype {
	this := nonCharStringSubtype{}
	this.allowed = allowed
	this.values = values
	return this
}

//shallow:allow semantic subtype factory is kept consistently with the other subtype factories
func nonCharStringSubtypeFrom(allowed bool, values []enumerableType[string]) nonCharStringSubtype {
	return newNonCharStringSubtypeFromBoolEnumerableStrings(allowed, values)
}

func (n *nonCharStringSubtype) Allowed() bool {
	return n.allowed
}

func (n *nonCharStringSubtype) Values() []enumerableType[string] {
	return n.values
}
