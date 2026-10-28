// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package text

import "strings"

var sourceNewlineNormalizer = strings.NewReplacer("\r\n", "\n", "\r", "\n")

func NormalizeSource(source string) string {
	source = strings.TrimPrefix(source, "\uFEFF")
	return sourceNewlineNormalizer.Replace(source)
}

// TextDocumentFromText creates a TextDocument from the given text string.
func TextDocumentFromText(text string) TextDocument {
	return NewStringTextDocument(text)
}
