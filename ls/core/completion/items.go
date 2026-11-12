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

package completion

import (
	"sort"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type itemSet struct {
	byLabel map[string]protocol.CompletionItem
}

func newItemSet() *itemSet {
	return &itemSet{byLabel: make(map[string]protocol.CompletionItem)}
}

func (s *itemSet) add(item protocol.CompletionItem) {
	if item.Label == "" {
		return
	}
	if _, found := s.byLabel[item.Label]; !found {
		s.byLabel[item.Label] = item
	}
}

func (s *itemSet) items() []protocol.CompletionItem {
	return s.itemsRanked(nil)
}

func (s *itemSet) itemsRanked(compatible func(protocol.CompletionItem) bool) []protocol.CompletionItem {
	labels := make([]string, 0, len(s.byLabel))
	for label := range s.byLabel {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	items := make([]protocol.CompletionItem, 0, len(labels))
	if compatible == nil {
		for _, label := range labels {
			items = append(items, s.byLabel[label])
		}
		return items
	}
	for _, label := range labels {
		item := s.byLabel[label]
		if compatible(item) {
			items = append(items, item)
		}
	}
	for _, label := range labels {
		item := s.byLabel[label]
		if !compatible(item) {
			items = append(items, item)
		}
	}
	return items
}

func isGeneratedName(name string) bool {
	return strings.HasPrefix(name, "$")
}

func completionItemKind(kind model.SymbolKind) protocol.CompletionItemKind {
	switch kind {
	case model.SymbolKindFunction:
		return protocol.CompletionItemKindFunction
	case model.SymbolKindConstant:
		return protocol.CompletionItemKindConstant
	case model.SymbolKindVariable, model.SymbolKindParemeter:
		return protocol.CompletionItemKindVariable
	case model.SymbolKindType:
		return protocol.CompletionItemKindClass
	default:
		return protocol.CompletionItemKindText
	}
}

func keywordItem(label string) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label:      label,
		Kind:       protocol.NewOptional(protocol.CompletionItemKindKeyword),
		InsertText: protocol.NewOptional(label),
	}
}
