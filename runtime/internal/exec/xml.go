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

package exec

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func filterXML(source any, patterns []bir.XMLNamePattern) *values.XMLSequence {
	xmlValue, ok := source.(values.XMLValue)
	if !ok {
		panic(fmt.Sprintf("invariant violation: XMLFilter source is not an XMLValue (got %T)", source))
	}
	matches := make([]values.XMLValue, 0)
	for _, item := range xmlValue.IterItems() {
		element, ok := item.(*values.XMLElement)
		if !ok {
			continue
		}
		for _, pattern := range patterns {
			if xmlNamePatternMatches(element, pattern) {
				matches = append(matches, element)
				break
			}
		}
	}
	return values.NewNormalizedXMLSequence(matches)
}

func xmlNamePatternMatches(element *values.XMLElement, pattern bir.XMLNamePattern) bool {
	switch pattern.Kind {
	case bir.XMLNamePatternKindWildCard:
		return true
	case bir.XMLNamePatternKindIdentifier:
		return element.NamespaceURI == "" && element.LocalName == pattern.Identifier
	case bir.XMLNamePatternKindQualifiedIdentifier:
		return element.NamespaceURI == pattern.NamespaceURI && element.LocalName == pattern.Identifier
	case bir.XMLNamePatternKindPrefix:
		return element.NamespaceURI == pattern.NamespaceURI
	default:
		panic("invariant violation: unsupported XML name pattern kind")
	}
}
