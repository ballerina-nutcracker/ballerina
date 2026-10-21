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
	return strings.ToValidUTF8(string(data), "\uFFFD")
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
