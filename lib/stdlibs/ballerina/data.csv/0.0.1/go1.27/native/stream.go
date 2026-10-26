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

package native

import (
	"strings"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/values"
)

type charset int

const (
	charsetUTF8 charset = iota
	charsetASCII
	charsetLatin1
)

// charsetNames maps the Java charset names and aliases this port decodes, compared case-insensitively.
var charsetNames = map[string]charset{
	"utf-8": charsetUTF8, "utf8": charsetUTF8, "unicode-1-1-utf-8": charsetUTF8,
	"us-ascii": charsetASCII, "ascii": charsetASCII, "iso646-us": charsetASCII, "us": charsetASCII,
	"cp367": charsetASCII, "ibm367": charsetASCII, "646": charsetASCII, "default": charsetASCII,
	"iso-8859-1": charsetLatin1, "iso8859_1": charsetLatin1, "iso_8859_1": charsetLatin1,
	"iso_8859-1": charsetLatin1, "iso8859-1": charsetLatin1, "8859_1": charsetLatin1, "latin1": charsetLatin1,
	"l1": charsetLatin1, "cp819": charsetLatin1, "ibm819": charsetLatin1, "ibm-819": charsetLatin1,
	"819": charsetLatin1, "csisolatin1": charsetLatin1, "iso-ir-100": charsetLatin1,
}

// decoder turns bytes into characters; malformed input becomes U+FFFD, as Java's InputStreamReader does.
type decoder struct {
	charset charset
}

func newDecoder(encoding string) (*decoder, bool) {
	cs, ok := charsetNames[strings.ToLower(encoding)]
	return &decoder{charset: cs}, ok
}

func (d *decoder) decodeAll(data []byte) string {
	switch d.charset {
	case charsetASCII:
		return strings.Map(func(r rune) rune {
			if r >= utf8.RuneSelf {
				return utf8.RuneError
			}
			return r
		}, latin1String(data))
	case charsetLatin1:
		return latin1String(data)
	}
	return decodeUTF8(data)
}

// decodeUTF8 replaces each malformed sequence with one U+FFFD, grouping bytes the way Java's UTF-8
// decoder does.
func decodeUTF8(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	var sb strings.Builder
	sb.Grow(len(data))
	for len(data) > 0 {
		if r, size := utf8.DecodeRune(data); r != utf8.RuneError || size > 1 {
			sb.Write(data[:size])
			data = data[size:]
			continue
		}
		sb.WriteRune(utf8.RuneError)
		data = data[javaMalformedLength(data):]
	}
	return sb.String()
}

// javaMalformedLength is the number of leading bytes of an invalid sequence that Java's decoder replaces
// with one U+FFFD: the lead byte and the continuation bytes that can still extend it. A complete
// surrogate encoding counts as one sequence.
func javaMalformedLength(data []byte) int {
	need, lo, hi := 0, byte(0x80), byte(0xBF)
	switch lead := data[0]; {
	case lead >= 0xC2 && lead <= 0xDF:
		need = 1
	case lead == 0xE0:
		need, lo = 2, 0xA0
	case lead >= 0xE1 && lead <= 0xEF:
		need = 2
	case lead == 0xF0:
		need, lo = 3, 0x90
	case lead >= 0xF1 && lead <= 0xF3:
		need = 3
	case lead == 0xF4:
		need, hi = 3, 0x8F
	default:
		return 1
	}
	n := 1
	for n <= need && n < len(data) && data[n] >= lo && data[n] <= hi {
		lo, hi = 0x80, 0xBF
		n++
	}
	return n
}

func latin1String(data []byte) string {
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}

// readBlock returns the next block of a byte-block stream, nil at its end, or the error it completed
// with.
func readBlock(source *values.Stream) ([]byte, *values.Error, bool) {
	switch next := source.Next().(type) {
	case *values.Error:
		return nil, next, false
	case *values.Map:
		block, _ := next.Get("value")
		list, _ := block.(*values.List)
		return list.ToByteSlice(), nil, true
	}
	return nil, nil, false
}

// readAllBlocks reads a byte-block stream to its end. An error the stream completes with is returned as
// it is, without wrapping.
func readAllBlocks(source *values.Stream) ([]byte, *values.Error) {
	var content []byte
	for {
		block, sourceErr, ok := readBlock(source)
		if sourceErr != nil {
			closeSource(source)
			return nil, sourceErr
		}
		if !ok {
			closeSource(source)
			return content, nil
		}
		content = append(content, block...)
	}
}

func closeSource(source *values.Stream) values.BalValue {
	if source.Close == nil {
		return nil
	}
	return source.Close()
}
