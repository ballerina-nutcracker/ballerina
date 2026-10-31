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

import "strings"

const (
	// nameReplacement is what Java's String decoder puts in place of a malformed UTF-8 sequence in a name.
	nameReplacement = '�'
	// commentReplacement is what the decoder of Apache Commons Compress puts there in a comment.
	commentReplacement = '?'
)

// cp437HighHalf holds the code points of the bytes 0x80 to 0xFF in CP437; the lower half is ASCII.
var cp437HighHalf = []rune("Çüéâäàåçêëèï" +
	"îìÄÅÉæÆôöòûù" +
	"ÿÖÜ¢£¥₧ƒáíóú" +
	"ñÑªº¿⌐¬½¼¡«»" +
	"░▒▓│┤╡╢╖╕╣║╗" +
	"╝╜╛┐└┴┬├─┼╞╟" +
	"╚╔╩╦╠═╬╧╨╤╥╙" +
	"╘╒╓╫╪┘┌█▄▌▐▀" +
	"αßΓπΣσµτΦΘΩδ" +
	"∞φε∩≡±≥≤⌠⌡÷≈" +
	"°∙·√ⁿ²■ ")

// entryName decodes the raw name of an entry: UTF-8 when general purpose bit 11 is set, CP437 otherwise.
func entryName(raw []byte, flags uint16) string {
	if flags&flagUTF8 != 0 {
		return decodeUTF8(raw, nameReplacement)
	}
	return decodeCP437(raw)
}

func decodeCP437(raw []byte) string {
	var b strings.Builder
	b.Grow(len(raw))
	for _, c := range raw {
		if c < 0x80 {
			b.WriteByte(c)
		} else {
			b.WriteRune(cp437HighHalf[c-0x80])
		}
	}
	return b.String()
}

func isContinuation(b byte) bool {
	return b&0xc0 == 0x80
}

// decodeUTF8 decodes like the JDK's UTF-8 decoder, which decides how many bytes one replacement stands
// for differently from Go: an encoded surrogate is one replacement, and a sequence cut short by the end of
// the input is one replacement however many of its bytes are there.
func decodeUTF8(src []byte, replacement rune) string {
	var b strings.Builder
	b.Grow(len(src))
	sp, sl := 0, len(src)
	for sp < sl {
		b1 := src[sp]
		sp++
		switch {
		case b1 < 0x80:
			b.WriteByte(b1)
		case b1>>5 == 0x6 && b1&0x1e != 0:
			if sp >= sl {
				b.WriteRune(replacement)
				return b.String()
			}
			if !isContinuation(src[sp]) {
				b.WriteRune(replacement)
				continue
			}
			b.WriteRune(rune(b1&0x1f)<<6 | rune(src[sp]&0x3f))
			sp++
		case b1>>4 == 0xe:
			if sp+1 < sl {
				b2, b3 := src[sp], src[sp+1]
				if malformed3(b1, b2) || !isContinuation(b3) {
					b.WriteRune(replacement)
					if !malformed3(b1, b2) {
						sp++
					}
					continue
				}
				sp += 2
				c := rune(b1&0x0f)<<12 | rune(b2&0x3f)<<6 | rune(b3&0x3f)
				if c >= 0xd800 && c <= 0xdfff {
					b.WriteRune(replacement)
				} else {
					b.WriteRune(c)
				}
				continue
			}
			b.WriteRune(replacement)
			if sp < sl && malformed3(b1, src[sp]) {
				continue
			}
			return b.String()
		case b1>>3 == 0x1e:
			if sp+2 < sl {
				b2, b3, b4 := src[sp], src[sp+1], src[sp+2]
				c := rune(b1&0x07)<<18 | rune(b2&0x3f)<<12 | rune(b3&0x3f)<<6 | rune(b4&0x3f)
				if !isContinuation(b2) || !isContinuation(b3) || !isContinuation(b4) || c < 0x10000 || c > 0x10ffff {
					b.WriteRune(replacement)
					sp += malformed4Length(b1, b2, b3) - 1
					continue
				}
				sp += 3
				b.WriteRune(c)
				continue
			}
			b.WriteRune(replacement)
			if b1 > 0xf4 || sp < sl && malformed4Second(b1, src[sp]) {
				continue
			}
			sp++
			if sp < sl && !isContinuation(src[sp]) {
				continue
			}
			return b.String()
		default:
			b.WriteRune(replacement)
		}
	}
	return b.String()
}

func malformed3(b1, b2 byte) bool {
	return b1 == 0xe0 && b2&0xe0 == 0x80 || !isContinuation(b2)
}

func malformed4Second(b1, b2 byte) bool {
	return b1 == 0xf0 && (b2 < 0x90 || b2 > 0xbf) || b1 == 0xf4 && b2&0xf0 != 0x80 || !isContinuation(b2)
}

// malformed4Length is how many bytes of a four byte sequence one replacement stands for.
func malformed4Length(b1, b2, b3 byte) int {
	if b1 > 0xf4 || malformed4Second(b1, b2) {
		return 1
	}
	if !isContinuation(b3) {
		return 2
	}
	return 3
}
