// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	"strings"
)

const tspecials = `()<>@,;:\"/[]?=`

type headerParam struct {
	name  string
	value string
}

type mediaType struct {
	primaryType string
	subType     string
	suffix      string
	params      []headerParam
}

type contentDisposition struct {
	disposition string
	name        string
	fileName    string
	params      []headerParam
}

// parseMediaType follows javax.activation.MimeType, which jBallerina uses: type and
// sub-type are required RFC 2045 tokens, lowercased, and the full sub-type (including
// any `+suffix`) is kept. Params keep their source order.
func parseMediaType(s string) (mediaType, bool) {
	slash := strings.IndexByte(s, '/')
	semi := strings.IndexByte(s, ';')
	if slash < 0 || (semi >= 0 && semi < slash) {
		return mediaType{}, false
	}
	end := len(s)
	if semi >= 0 {
		end = semi
	}
	mt := mediaType{
		primaryType: strings.ToLower(strings.TrimSpace(s[:slash])),
		subType:     strings.ToLower(strings.TrimSpace(s[slash+1 : end])),
	}
	if !isToken(mt.primaryType) || !isToken(mt.subType) {
		return mediaType{}, false
	}
	if i := strings.LastIndexByte(mt.subType, '+'); i >= 0 {
		mt.suffix = mt.subType[i+1:]
	}
	if semi >= 0 {
		params, ok := parseMediaTypeParams(s[semi:])
		if !ok {
			return mediaType{}, false
		}
		mt.params = params
	}
	return mt, true
}

func parseMediaTypeParams(s string) ([]headerParam, bool) {
	var params []headerParam
	i := 0
	for {
		i = skipSpace(s, i)
		if i >= len(s) {
			return params, true
		}
		if s[i] != ';' {
			return nil, false
		}
		i = skipSpace(s, i+1)
		if i >= len(s) {
			return params, true
		}
		start := i
		for i < len(s) && isTokenChar(s[i]) {
			i++
		}
		name := strings.ToLower(s[start:i])
		i = skipSpace(s, i)
		if name == "" || i >= len(s) || s[i] != '=' {
			return nil, false
		}
		value, next, ok := parseParamValue(s, skipSpace(s, i+1))
		if !ok {
			return nil, false
		}
		params = append(params, headerParam{name: name, value: value})
		i = next
	}
}

func parseParamValue(s string, i int) (string, int, bool) {
	if i < len(s) && s[i] == '"' {
		var sb strings.Builder
		for i++; i < len(s); i++ {
			switch s[i] {
			case '"':
				return sb.String(), i + 1, true
			case '\\':
				if i+1 < len(s) {
					i++
				}
			}
			sb.WriteByte(s[i])
		}
		return "", 0, false
	}
	start := i
	for i < len(s) && isTokenChar(s[i]) {
		i++
	}
	return s[start:i], i, i > start
}

// parseContentDisposition splits on `;` and `=` the way jBallerina does: the
// disposition keeps its case, `name` and `filename` are unquoted, and every other
// param is kept verbatim.
func parseContentDisposition(s string) contentDisposition {
	segments := strings.Split(s, ";")
	cd := contentDisposition{disposition: strings.TrimSpace(segments[0])}
	for _, segment := range segments[1:] {
		key, value, found := strings.Cut(segment, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !found || key == "" {
			continue
		}
		switch strings.ToLower(key) {
		case "name":
			cd.name = unquote(value)
		case "filename":
			cd.fileName = unquote(value)
		default:
			cd.params = append(cd.params, headerParam{name: key, value: value})
		}
	}
	return cd
}

func (cd contentDisposition) String() string {
	if cd.disposition == "" {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(cd.disposition)
	if cd.name != "" {
		sb.WriteString(";name=" + quote(cd.name))
	}
	if cd.fileName != "" {
		sb.WriteString(";filename=" + quote(cd.fileName))
	}
	for _, p := range cd.params {
		sb.WriteString(";" + p.name + "=" + p.value)
	}
	return sb.String()
}

func quote(s string) string {
	if !strings.HasPrefix(s, `"`) {
		s = `"` + s
	}
	if !strings.HasSuffix(s, `"`) || len(s) == 1 {
		s += `"`
	}
	return s
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func isToken(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if !isTokenChar(s[i]) {
			return false
		}
	}
	return true
}

func isTokenChar(c byte) bool {
	return c > 0x20 && c < 0x7f && !strings.ContainsRune(tspecials, rune(c))
}

func skipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}
