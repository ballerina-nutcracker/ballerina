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

// mediaTypeError carries javax.activation's MimeTypeParseException text verbatim.
type mediaTypeError string

func (e mediaTypeError) Error() string {
	return string(e)
}

func (mt mediaType) param(name string) string {
	for _, p := range mt.params {
		if p.name == name {
			return p.value
		}
	}
	return ""
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
