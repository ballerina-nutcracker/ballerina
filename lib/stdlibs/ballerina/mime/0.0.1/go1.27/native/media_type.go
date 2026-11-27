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
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
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

// mediaTypeError carries javax.activation's MimeTypeParseException text verbatim.
type mediaTypeError string

func (e mediaTypeError) Error() string {
	return string(e)
}

func (mt mediaType) param(name string) string {
	v, _ := mt.lookup(name)
	return v
}

func (mt mediaType) lookup(name string) (string, bool) {
	for _, p := range mt.params {
		if p.name == name {
			return p.value, true
		}
	}
	return "", false
}

// extendedParam returns the RFC 2231 form of a parameter (`name*`, or the continuations
// `name*0`, `name*1*`, ...) decoded; ok is false when it is absent or malformed.
func (mt mediaType) extendedParam(name string) (string, bool) {
	if v, ok := mt.lookup(name + "*"); ok {
		return decode2231(v, true)
	}
	var b strings.Builder
	for n := 0; ; n++ {
		key := name + "*" + strconv.Itoa(n)
		if v, ok := mt.lookup(key); ok {
			if !isASCII(v) {
				return "", false
			}
			b.WriteString(v)
			continue
		}
		v, ok := mt.lookup(key + "*")
		if !ok {
			return b.String(), n > 0 && !mt.hasSegmentAfter(name, n)
		}
		part, ok := decode2231(v, n == 0)
		if !ok {
			return "", false
		}
		b.WriteString(part)
	}
}

// hasSegmentAfter reports whether a continuation of name is numbered above the missing index n.
func (mt mediaType) hasSegmentAfter(name string, n int) bool {
	prefix := name + "*"
	for _, p := range mt.params {
		index, ok := strings.CutPrefix(p.name, prefix)
		if !ok {
			continue
		}
		if m, err := strconv.Atoi(strings.TrimSuffix(index, "*")); err == nil && m > n {
			return true
		}
	}
	return false
}

// decode2231 percent-decodes an RFC 2231 value, first dropping its `charset'language'`
// prefix when it has one. Only attribute-chars and escapes, decoding to ASCII, are accepted.
func decode2231(v string, prefixed bool) (string, bool) {
	if prefixed {
		parts := strings.SplitN(v, "'", 3)
		if len(parts) != 3 || !isAttributeChars(parts[0]) || !isLanguageTag(parts[1]) {
			return "", false
		}
		v = parts[2]
	}
	for i := range len(v) {
		if c := v[i]; c != '%' && !isAttributeChar(c) {
			return "", false
		}
	}
	decoded, err := url.PathUnescape(v)
	if err != nil {
		return "", false
	}
	if !isASCII(decoded) {
		return "", false
	}
	return decoded, true
}

func isAttributeChar(c byte) bool {
	return isTokenChar(c) && c != '*' && c != '\'' && c != '%'
}

func isAttributeChars(s string) bool {
	for i := range len(s) {
		if !isAttributeChar(s[i]) {
			return false
		}
	}
	return true
}

// isLanguageTag accepts an empty tag or alphanumeric subtags joined by hyphens, as RFC 2231's language field allows.
func isLanguageTag(s string) bool {
	if s == "" {
		return true
	}
	for sub := range strings.SplitSeq(s, "-") {
		if sub == "" || len(sub) > 8 {
			return false
		}
		for i := range len(sub) {
			if c := sub[i]; !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9') {
				return false
			}
		}
	}
	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// parseMediaType follows javax.activation.MimeType, which jBallerina uses: type and
// sub-type are required RFC 2045 tokens, lowercased, and the full sub-type (including
// any `+suffix`) is kept. Params keep their source order.
func parseMediaType(s string) (mediaType, error) {
	slash := strings.IndexByte(s, '/')
	semi := strings.IndexByte(s, ';')
	if slash < 0 || (semi >= 0 && semi < slash) {
		return mediaType{}, mediaTypeError("Unable to find a sub type.")
	}
	end := len(s)
	if semi >= 0 {
		end = semi
	}
	mt := mediaType{
		primaryType: strings.ToLower(strings.TrimSpace(s[:slash])),
		subType:     strings.ToLower(strings.TrimSpace(s[slash+1 : end])),
	}
	if semi >= 0 {
		params, err := parseMediaTypeParams(s[semi:])
		if err != nil {
			return mediaType{}, err
		}
		mt.params = params
	}
	if !isToken(mt.primaryType) {
		return mediaType{}, mediaTypeError("Primary type is invalid.")
	}
	if !isToken(mt.subType) {
		return mediaType{}, mediaTypeError("Sub type is invalid.")
	}
	if i := strings.LastIndexByte(mt.subType, '+'); i >= 0 {
		mt.suffix = mt.subType[i+1:]
	}
	return mt, nil
}

// parseMediaTypeParams follows javax.activation.MimeTypeParameterList.parse, including
// its acceptance of an empty parameter name.
func parseMediaTypeParams(s string) ([]headerParam, error) {
	var params []headerParam
	i := skipSpace(s, 0)
	for i < len(s) && s[i] == ';' {
		i = skipSpace(s, i+1)
		if i >= len(s) {
			return params, nil
		}
		start := i
		for i < len(s) && isTokenChar(s[i]) {
			i++
		}
		name := strings.ToLower(s[start:i])
		i = skipSpace(s, i)
		if i >= len(s) || s[i] != '=' {
			return nil, mediaTypeError("Couldn't find the '=' that separates a parameter name from its value.")
		}
		i = skipSpace(s, i+1)
		if i >= len(s) {
			return nil, mediaTypeError(fmt.Sprintf("Couldn't find a value for parameter named %s", name))
		}
		value, next, err := parseParamValue(s, i)
		if err != nil {
			return nil, err
		}
		params = setParam(params, name, value)
		i = skipSpace(s, next)
	}
	if i < len(s) {
		return nil, mediaTypeError("More characters encountered in input than expected.")
	}
	return params, nil
}

func setParam(params []headerParam, name, value string) []headerParam {
	for i := range params {
		if params[i].name == name {
			params[i].value = value
			return params
		}
	}
	return append(params, headerParam{name: name, value: value})
}

func parseParamValue(s string, i int) (string, int, error) {
	if s[i] == '"' {
		var sb strings.Builder
		for i++; i < len(s); i++ {
			switch s[i] {
			case '"':
				return sb.String(), i + 1, nil
			case '\\':
				if i+1 < len(s) {
					i++
				}
			}
			sb.WriteByte(s[i])
		}
		return "", 0, mediaTypeError("Encountered unterminated quoted parameter value.")
	}
	if !isTokenChar(s[i]) {
		return "", 0, mediaTypeError(fmt.Sprintf("Unexpected character encountered at index %d", i))
	}
	start := i
	for i < len(s) && isTokenChar(s[i]) {
		i++
	}
	return s[start:i], i, nil
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
