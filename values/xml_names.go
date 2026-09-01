// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package values

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	XMLNamespaceURI   = "http://www.w3.org/XML/1998/namespace"
	XMLNSNamespaceURI = "http://www.w3.org/2000/xmlns/"
)

func ExpandedXMLName(namespaceURI, localName string) string {
	if namespaceURI == "" {
		return localName
	}
	return "{" + namespaceURI + "}" + localName
}

func ParseExpandedXMLName(name string) (string, string, error) {
	if err := ValidateXMLCharacters(name); err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(name, "{") {
		if !isNCName(name) {
			return "", "", fmt.Errorf("invalid XML expanded name %q", name)
		}
		return "", name, nil
	}
	close := strings.IndexByte(name, '}')
	if close <= 1 || close == len(name)-1 || strings.ContainsAny(name[1:close], "{}") || strings.ContainsAny(name[close+1:], "{}") || strings.Count(name, "{") != 1 || strings.Count(name, "}") != 1 || !isNCName(name[close+1:]) {
		return "", "", fmt.Errorf("invalid XML expanded name %q", name)
	}
	return name[1:close], name[close+1:], nil
}

func ValidateXMLCharacters(s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("invalid UTF-8 in XML value")
	}
	for _, r := range s {
		if r != 0x9 && r != 0xA && r != 0xD && (r < 0x20 || r > 0xD7FF && r < 0xE000 || r > 0xFFFD && r < 0x10000 || r > 0x10FFFF) {
			return fmt.Errorf("illegal XML character U+%04X", r)
		}
	}
	return nil
}

func isNameStart(r rune, allowColon bool) bool {
	return r == '_' || allowColon && r == ':' || unicode.IsLetter(r) || r >= 0xC0 && r <= 0xD6 || r >= 0xD8 && r <= 0xF6 || r >= 0xF8 && r <= 0x2FF || r >= 0x370 && r <= 0x37D || r >= 0x37F && r <= 0x1FFF || r >= 0x200C && r <= 0x200D || r >= 0x2070 && r <= 0x218F || r >= 0x2C00 && r <= 0x2FEF || r >= 0x3001 && r <= 0xD7FF || r >= 0xF900 && r <= 0xFDCF || r >= 0xFDF0 && r <= 0xFFFD || r >= 0x10000 && r <= 0xEFFFF
}

func isXMLName(name string, allowColon bool) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !isNameStart(r, allowColon) {
				return false
			}
			continue
		}
		if !isNameStart(r, allowColon) && r != '-' && r != '.' && !unicode.IsDigit(r) && r != 0xB7 && (r < 0x300 || r > 0x36F) && (r < 0x203F || r > 0x2040) {
			return false
		}
	}
	return true
}

func isNCName(name string) bool { return !strings.ContainsRune(name, ':') && isXMLName(name, false) }

func ValidateXMLComment(content string) error {
	if err := ValidateXMLCharacters(content); err != nil {
		return err
	}
	if strings.Contains(content, "--") || strings.HasSuffix(content, "-") {
		return fmt.Errorf("XML comment body must not contain '--' or end with '-'")
	}
	return nil
}

func ValidateXMLProcessingInstruction(target, content string) error {
	if err := ValidateXMLCharacters(target + content); err != nil {
		return err
	}
	if !isXMLName(target, true) || strings.EqualFold(target, "xml") {
		return fmt.Errorf("invalid or reserved XML processing-instruction target %q", target)
	}
	if strings.Contains(content, "?>") {
		return fmt.Errorf("XML processing-instruction content must not contain '?>'")
	}
	return nil
}

func validateNamespaceBinding(prefix, uri string) error {
	if prefix != "" && !isNCName(prefix) {
		return fmt.Errorf("invalid namespace prefix %q", prefix)
	}
	if prefix == "xmlns" || uri == XMLNSNamespaceURI {
		return fmt.Errorf("reserved xmlns namespace binding")
	}
	if prefix == "xml" && uri != XMLNamespaceURI || prefix != "xml" && uri == XMLNamespaceURI {
		return fmt.Errorf("invalid xml namespace binding")
	}
	if prefix != "" && uri == "" {
		return fmt.Errorf("prefixed namespace declaration cannot have an empty URI")
	}
	return ValidateXMLCharacters(uri)
}
