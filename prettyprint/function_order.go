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

package prettyprint

import (
	"cmp"
	"strconv"
	"strings"
)

const generatedNamePrefix = "$"

// CompareFunctionPrintOrder orders user written functions before the compiler
// generated ("$"-prefixed) ones. Generated functions are grouped by kind (the
// name without its trailing index) and ordered numerically by index within a
// kind. Two user written functions compare equal so a stable sort keeps them in
// the order the printer received them.
func CompareFunctionPrintOrder(a, b string) int {
	aGenerated := strings.HasPrefix(a, generatedNamePrefix)
	bGenerated := strings.HasPrefix(b, generatedNamePrefix)
	switch {
	case !aGenerated && !bGenerated:
		return 0
	case !aGenerated:
		return -1
	case !bGenerated:
		return 1
	}
	aKind, aIndex := splitGeneratedName(a)
	bKind, bIndex := splitGeneratedName(b)
	return cmp.Or(cmp.Compare(aKind, bKind), cmp.Compare(aIndex, bIndex))
}

// splitGeneratedName splits name into its kind and the index given by its
// trailing decimal digits. The index is -1 when name has no trailing digits.
func splitGeneratedName(name string) (string, int) {
	kind := strings.TrimRightFunc(name, func(r rune) bool { return r >= '0' && r <= '9' })
	if kind == name {
		return name, -1
	}
	index, err := strconv.Atoi(name[len(kind):])
	if err != nil {
		return name, -1
	}
	return kind, index
}
