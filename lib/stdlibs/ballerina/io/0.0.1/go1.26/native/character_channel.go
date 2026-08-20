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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type characterChannelTypes struct {
	strArrTy   semtypes.SemType
	strMapTy   semtypes.SemType
	strMapAtom *semtypes.MappingAtomicType
	jsonListTy semtypes.SemType
	jsonMapTy  semtypes.SemType

	lineStreamTy   semtypes.SemType
	lineRecordTy   semtypes.SemType
	lineRecordAtom *semtypes.MappingAtomicType
}

func charChannelClosedError() values.BalValue {
	return fileIOError("Character channel is already closed.")
}

func lookupCharset(charset string) (encoding.Encoding, error) {
	enc, err := ianaindex.IANA.Encoding(charset)
	if err != nil || enc == nil {
		return nil, fmt.Errorf("unsupported encoding type %s", charset)
	}
	return enc, nil
}

func charReaderOf(self *values.Object) *bufio.Reader {
	v, _ := self.Get("$charReader")
	r, _ := v.(*bufio.Reader)
	return r
}

func byteChannelOf(self *values.Object) (*values.Object, bool) {
	v, ok := self.Get("$byteChannel")
	if !ok {
		return nil, false
	}
	obj, ok := v.(*values.Object)
	return obj, ok
}

// countingWriter tracks the number of bytes written to the underlying
// writer, so writeEncoded can report the destination byte count for a
// single call even though the encoder itself is shared across calls.
type countingWriter struct {
	w io.Writer
	n int
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += n
	return n, err
}

// charWriterState is the persistent per-channel encoding writer. The
// encoder must be created once per channel and reused across write() calls
// (like charReaderOf's decoder), not recreated per call, since a stateful
// encoding (e.g. UTF-16) would otherwise emit its byte-order mark on every
// single write instead of once for the whole channel.
type charWriterState struct {
	writer  *transform.Writer
	counter *countingWriter
}

func charWriterOf(self *values.Object) (*charWriterState, bool) {
	v, ok := self.Get("$charWriter")
	if !ok {
		return nil, false
	}
	st, ok := v.(*charWriterState)
	return st, ok
}

func eofReached(self *values.Object) bool {
	v, ok := self.Get("$eof")
	if !ok {
		return false
	}
	eof, _ := v.(bool)
	return eof
}

// drainChars reads the remaining decoded content of the channel and marks it
// as fully consumed, mirroring jBallerina, where whole-content reads leave the
// channel at EOF.
func drainChars(self *values.Object, r *bufio.Reader) (string, error) {
	data, err := io.ReadAll(r)
	self.Put("$eof", true)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// closeUnderlyingByteChannel closes the wrapped byte channel exactly once,
// mirroring jBallerina, where closing a character channel chains to the byte
// channel beneath it.
func closeUnderlyingByteChannel(self *values.Object) error {
	byteCh, ok := byteChannelOf(self)
	if !ok || isClosed(byteCh) {
		return nil
	}
	markClosed(byteCh)
	if closer, ok := closerOf(byteCh); ok {
		return closer.closeOnce()
	}
	if writer, ok := writerOf(byteCh); ok {
		return writer.Close()
	}
	return nil
}

// writeEncoded encodes content using the channel's persistent encoder and
// writes it to the underlying byte channel, returning the number of
// destination bytes written for this call.
func writeEncoded(self *values.Object, content string) (int, values.BalValue) {
	st, ok := charWriterOf(self)
	if !ok {
		return 0, fileIOError("Byte channel is not initialized")
	}
	before := st.counter.n
	if _, err := st.writer.Write([]byte(content)); err != nil {
		return 0, fileIOError("error occurred while writing characters to the channel. " + err.Error())
	}
	return st.counter.n - before, nil
}

// channelProperties returns the parsed .properties content of the channel,
// draining and parsing it on first use and caching the result, mirroring
// jBallerina's per-channel Properties cache.
func channelProperties(self *values.Object, r *bufio.Reader) ([]propertyEntry, error) {
	if v, ok := self.Get("$props"); ok {
		props, _ := v.([]propertyEntry)
		return props, nil
	}
	content, err := drainChars(self, r)
	if err != nil {
		return nil, err
	}
	props := parseProperties(content)
	self.Put("$props", props)
	return props, nil
}

type propertyEntry struct {
	key   string
	value string
}

// parseProperties parses content in java.util.Properties format: logical
// lines with backslash continuations, '#'/'!' comments, '='/':'/ whitespace
// key separators, and \t \n \f \r \\ \uXXXX escapes.
func parseProperties(content string) []propertyEntry {
	var entries []propertyEntry
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimLeft(lines[i], " \t\f")
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		for hasOddTrailingBackslashes(line) && i+1 < len(lines) {
			i++
			line = line[:len(line)-1] + strings.TrimLeft(lines[i], " \t\f")
		}
		key, value := splitPropertyLine(line)
		entries = append(entries, propertyEntry{key: unescapeProperty(key), value: unescapeProperty(value)})
	}
	return entries
}

func hasOddTrailingBackslashes(line string) bool {
	count := 0
	for i := len(line) - 1; i >= 0 && line[i] == '\\'; i-- {
		count++
	}
	return count%2 == 1
}

// splitPropertyLine splits a logical property line into raw key and value: the
// key ends at the first unescaped '=', ':', or whitespace; a whitespace
// terminator may be followed by one optional '=' or ':' before the value, and
// the value's leading whitespace is trimmed.
func splitPropertyLine(line string) (string, string) {
	keyEnd := len(line)
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '\\' {
			i++
			continue
		}
		if c == '=' || c == ':' || c == ' ' || c == '\t' || c == '\f' {
			keyEnd = i
			break
		}
	}
	key := line[:keyEnd]
	rest := strings.TrimLeft(line[keyEnd:], " \t\f")
	if rest != "" && (rest[0] == '=' || rest[0] == ':') {
		rest = strings.TrimLeft(rest[1:], " \t\f")
	}
	return key, rest
}

// peekSurrogateEscape parses a `\uXXXX` escape starting at s[i] and reports
// whether it is present, for recombining a `\uXXXX\uXXXX` surrogate pair.
func peekSurrogateEscape(s string, i int) (uint16, bool) {
	if i+5 >= len(s) || s[i] != '\\' || s[i+1] != 'u' {
		return 0, false
	}
	code, err := strconv.ParseUint(s[i+2:i+6], 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(code), true
}

func unescapeProperty(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			sb.WriteByte(c)
			continue
		}
		i++
		switch s[i] {
		case 't':
			sb.WriteByte('\t')
		case 'n':
			sb.WriteByte('\n')
		case 'f':
			sb.WriteByte('\f')
		case 'r':
			sb.WriteByte('\r')
		case 'u':
			if i+4 < len(s) {
				if code, err := strconv.ParseUint(s[i+1:i+5], 16, 16); err == nil {
					i += 4
					if utf16.IsSurrogate(rune(code)) {
						if low, ok := peekSurrogateEscape(s, i+1); ok {
							if r := utf16.DecodeRune(rune(code), rune(low)); r != unicode.ReplacementChar {
								sb.WriteRune(r)
								i += 6
								continue
							}
						}
					}
					sb.WriteRune(rune(code))
					continue
				}
			}
			sb.WriteByte('u')
		default:
			sb.WriteByte(s[i])
		}
	}
	return sb.String()
}

// writePropertiesComment appends a `#`-prefixed comment header to sb,
// matching jBallerina's actual output: characters above Latin-1 (0xFF) are
// \uXXXX-escaped per UTF-16 code unit (so surrogate pairs render as two
// escapes), while embedded '\n'/'\r' start a new '#'-prefixed line instead
// of corrupting the properties format, unlike escapeProperty's key/value
// rules which escape the whole Latin-1 range.
func writePropertiesComment(sb *strings.Builder, comment string) {
	units := utf16.Encode([]rune(comment))
	sb.WriteByte('#')
	for i := 0; i < len(units); i++ {
		c := units[i]
		switch {
		case c > 0xFF:
			fmt.Fprintf(sb, `\u%04X`, c)
		case c == '\n' || c == '\r':
			sb.WriteByte('\n')
			if c == '\r' && i+1 < len(units) && units[i+1] == '\n' {
				i++
			}
			if i == len(units)-1 || (units[i+1] != '#' && units[i+1] != '!') {
				sb.WriteByte('#')
			}
		default:
			sb.WriteByte(byte(c))
		}
	}
	sb.WriteByte('\n')
}

// escapeProperty escapes a key or value for java.util.Properties output:
// separators and specials are backslash-escaped, spaces are escaped in keys
// (and leading spaces in values), and chars outside 0x20..0x7E become \uXXXX.
func escapeProperty(s string, escapeAllSpaces bool) string {
	var sb strings.Builder
	for i, r := range s {
		switch r {
		case '\\', '=', ':', '#', '!':
			sb.WriteByte('\\')
			sb.WriteRune(r)
		case ' ':
			if escapeAllSpaces || i == 0 {
				sb.WriteByte('\\')
			}
			sb.WriteByte(' ')
		case '\t':
			sb.WriteString(`\t`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\f':
			sb.WriteString(`\f`)
		default:
			switch {
			case r >= 0x20 && r <= 0x7e:
				sb.WriteRune(r)
			case r <= 0xFFFF:
				fmt.Fprintf(&sb, `\u%04X`, r)
			default:
				for _, u := range utf16.Encode([]rune{r}) {
					fmt.Fprintf(&sb, `\u%04X`, u)
				}
			}
		}
	}
	return sb.String()
}

// xmlDoctypeString renders the `<!DOCTYPE ...>` line for the given root
// element name and io:XmlDoctype record, following jBallerina's precedence:
// internalSubset, then PUBLIC+SYSTEM, then PUBLIC, then SYSTEM.
func xmlDoctypeString(rootName string, doctype *values.Map) string {
	getField := func(name string) (string, bool) {
		v, ok := doctype.Get(name)
		if !ok || v == nil {
			return "", false
		}
		s, ok := v.(string)
		return s, ok
	}
	if internalSubset, ok := getField("internalSubset"); ok {
		return fmt.Sprintf("<!DOCTYPE %s %s>", rootName, internalSubset)
	}
	publicID, hasPublic := getField("public")
	systemID, hasSystem := getField("system")
	switch {
	case hasPublic && hasSystem:
		return fmt.Sprintf("<!DOCTYPE %s PUBLIC %q %q>", rootName, publicID, systemID)
	case hasPublic:
		return fmt.Sprintf("<!DOCTYPE %s PUBLIC %q>", rootName, publicID)
	case hasSystem:
		return fmt.Sprintf("<!DOCTYPE %s SYSTEM %q>", rootName, systemID)
	}
	return ""
}

// rootElementName returns the name of the root element of the given XML
// value, mirroring jBallerina's `<xml:Element>content` cast (an error for
// non-element content).
func rootElementName(content values.XMLValue) (string, error) {
	switch x := content.(type) {
	case *values.XMLElement:
		return x.Name, nil
	case *values.XMLSequence:
		if len(x.Children) == 1 {
			return rootElementName(x.Children[0])
		}
	}
	return "", fmt.Errorf("incompatible types: 'xml' cannot be cast to 'xml:Element'")
}

// characterChannelExterns implements the extern functions of both
// ReadableCharacterChannel and WritableCharacterChannel; each is registered
// as a named method rather than an inline closure, so the registration list
// stays a plain map from Ballerina method name to Go implementation.
type characterChannelExterns struct {
	rt    *runtime.Runtime
	types characterChannelTypes
}

func (e *characterChannelExterns) readableInitChannel(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	byteCh, _ := args[1].(*values.Object)
	charset, _ := args[2].(string)
	enc, err := lookupCharset(charset)
	if err != nil {
		// A Go error return panics in the interpreter, matching
		// jBallerina, where init throws for an unsupported charset.
		return nil, err
	}
	reader, ok := readerOf(byteCh)
	if !ok {
		return nil, fmt.Errorf("byte channel is not initialized")
	}
	self.Put("$charReader", bufio.NewReader(transform.NewReader(reader, enc.NewDecoder())))
	self.Put("$byteChannel", byteCh)
	return nil, nil
}

func (e *characterChannelExterns) readableRead(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	numberOfChars, _ := args[1].(int64)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	if eofReached(self) {
		return byteChannelEofError(), nil
	}
	r := charReaderOf(self)
	var sb strings.Builder
	for i := int64(0); i < numberOfChars; i++ {
		ch, _, err := r.ReadRune()
		if err == io.EOF {
			self.Put("$eof", true)
			break
		}
		if err != nil {
			return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
		}
		sb.WriteRune(ch)
	}
	return sb.String(), nil
}

func (e *characterChannelExterns) readableReadString(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	r := charReaderOf(self)
	content, err := drainChars(self, r)
	if err != nil {
		return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
	}
	return strings.Join(splitLines([]byte(content)), "\n"), nil
}

func (e *characterChannelExterns) readableReadAllLines(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	r := charReaderOf(self)
	content, err := drainChars(self, r)
	if err != nil {
		return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
	}
	lines := splitLines([]byte(content))
	items := make([]values.BalValue, len(lines))
	for i, line := range lines {
		items[i] = line
	}
	return values.NewList(e.types.strArrTy, semtypes.ToListAtomicType(e.rt.GetTypeEnv(), e.types.strArrTy), false, nil, 0, items), nil
}

func (e *characterChannelExterns) readableReadJson(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	r := charReaderOf(self)
	content, err := drainChars(self, r)
	if err != nil {
		return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
	}
	dec := json.NewDecoder(strings.NewReader(content))
	dec.UseNumber()
	var raw any
	if err := dec.Decode(&raw); err != nil {
		return fileIOError("error occurred while reading json from the channel. " + err.Error()), nil
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fileIOError("error occurred while reading json from the channel. trailing content after JSON value"), nil
	}
	return values.GoToBalValue(ctx.TypeCtx(), raw, e.types.jsonListTy, e.types.jsonMapTy), nil
}

func (e *characterChannelExterns) readableReadXml(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	r := charReaderOf(self)
	content, err := drainChars(self, r)
	if err != nil {
		return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
	}
	xmlVal, parseErr := values.ParseAsXMLValue(ctx.TypeCtx(), values.FromBytes([]byte(content)), values.XMLLenientMode)
	if parseErr != nil {
		return fileIOError("error occurred while reading xml from the channel. " + parseErr.Error()), nil
	}
	return xmlVal, nil
}

func (e *characterChannelExterns) readableReadProperty(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	key, _ := args[1].(string)
	defaultValue, _ := args[2].(string)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	r := charReaderOf(self)
	props, err := channelProperties(self, r)
	if err != nil {
		return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
	}
	for i := len(props) - 1; i >= 0; i-- {
		if props[i].key == key {
			return props[i].value, nil
		}
	}
	return defaultValue, nil
}

func (e *characterChannelExterns) readableReadAllProperties(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	r := charReaderOf(self)
	props, err := channelProperties(self, r)
	if err != nil {
		return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
	}
	entries := make([]values.MapEntry, len(props))
	for i, entry := range props {
		entries[i] = values.MapEntry{Key: entry.key, Value: entry.value}
	}
	return values.NewMap(e.types.strMapTy, e.types.strMapAtom, false, entries), nil
}

func (e *characterChannelExterns) readableLineStream(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	r := charReaderOf(self)
	next := func() values.BalValue {
		line, ok, readErr := readLineCRLF(r)
		if readErr != nil {
			_ = closeUnderlyingByteChannel(self)
			return fileIOError("error occurred while reading characters from the channel. " + readErr.Error())
		}
		if !ok {
			self.Put("$eof", true)
			if closeErr := closeUnderlyingByteChannel(self); closeErr != nil {
				return fileIOError("error occurred while closing the channel. " + closeErr.Error())
			}
			return nil
		}
		return values.NewMap(e.types.lineRecordTy, e.types.lineRecordAtom, false,
			[]values.MapEntry{{Key: "value", Value: line}})
	}
	closeFn := func() values.BalValue {
		if err := closeUnderlyingByteChannel(self); err != nil {
			return fileIOError("error occurred while closing the channel. " + err.Error())
		}
		return nil
	}
	return values.NewStream(e.types.lineStreamTy, next, closeFn), nil
}

func (e *characterChannelExterns) readableClose(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	markClosed(self)
	if err := closeUnderlyingByteChannel(self); err != nil {
		return fileIOError("error occurred while closing the channel. " + err.Error()), nil
	}
	return nil, nil
}

func (e *characterChannelExterns) writableInitChannel(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	byteCh, _ := args[1].(*values.Object)
	charset, _ := args[2].(string)
	enc, err := lookupCharset(charset)
	if err != nil {
		// A Go error return panics in the interpreter, matching
		// jBallerina, where init throws for an unsupported charset.
		return nil, err
	}
	writer, ok := writerOf(byteCh)
	if !ok {
		return nil, fmt.Errorf("byte channel is not initialized")
	}
	counter := &countingWriter{w: writer}
	self.Put("$charWriter", &charWriterState{
		writer:  transform.NewWriter(counter, enc.NewEncoder()),
		counter: counter,
	})
	self.Put("$byteChannel", byteCh)
	return nil, nil
}

func (e *characterChannelExterns) writableWrite(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	content, _ := args[1].(string)
	startOffset, _ := args[2].(int64)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	runes := []rune(content)
	if startOffset < 0 || startOffset > int64(len(runes)) {
		// jBallerina panics here too: CharBuffer.position throws
		// IllegalArgumentException for an out-of-range offset.
		return nil, fmt.Errorf("invalid start offset %d for content of %d characters", startOffset, len(runes))
	}
	n, errVal := writeEncoded(self, string(runes[startOffset:]))
	if errVal != nil {
		return errVal, nil
	}
	return int64(n), nil
}

func (e *characterChannelExterns) writableWriteJson(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	data, err := values.ToJSONByteArray(args[1])
	if err != nil {
		return fileIOError("error occurred while serializing json. " + err.Error()), nil
	}
	_, errVal := writeEncoded(self, string(data))
	return errVal, nil
}

func (e *characterChannelExterns) writableWriteXmlExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	content, _ := args[1].(values.XMLValue)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	writeContent := content.XMLString()
	if doctype, ok := args[2].(*values.Map); ok {
		rootName, err := rootElementName(content)
		if err != nil {
			// jBallerina panics here too: `<xml:Element>content` is a
			// failing cast for non-element content.
			return nil, err
		}
		if doctypeStr := xmlDoctypeString(rootName, doctype); doctypeStr != "" {
			writeContent = doctypeStr + "\n" + writeContent
		}
	}
	_, errVal := writeEncoded(self, writeContent)
	return errVal, nil
}

func (e *characterChannelExterns) writableWriteProperties(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	properties, _ := args[1].(*values.Map)
	comment, _ := args[2].(string)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	byteCh, _ := byteChannelOf(self)
	writer, ok := writerOf(byteCh)
	if !ok {
		return fileIOError("Byte channel is not initialized"), nil
	}
	var sb strings.Builder
	writePropertiesComment(&sb, comment)
	sb.WriteString("#" + e.rt.Platform().Time.Now().Format("Mon Jan 02 15:04:05 MST 2006") + "\n")
	for _, key := range properties.Keys() {
		value, _ := properties.Get(key)
		valueStr, _ := value.(string)
		sb.WriteString(escapeProperty(key, true) + "=" + escapeProperty(valueStr, false) + "\n")
	}
	// Properties output is escaped to plain ASCII, so it is written
	// directly to the byte channel, mirroring jBallerina, where
	// Properties.store bypasses the channel's charset.
	if _, err := writer.Write([]byte(sb.String())); err != nil {
		return fileIOError("error occurred while writing properties to the channel. " + err.Error()), nil
	}
	return nil, nil
}

func (e *characterChannelExterns) writableClose(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	self, _ := args[0].(*values.Object)
	if isClosed(self) {
		return charChannelClosedError(), nil
	}
	markClosed(self)
	if st, ok := charWriterOf(self); ok {
		if err := st.writer.Close(); err != nil {
			return fileIOError("error occurred while closing the channel. " + err.Error()), nil
		}
	}
	if err := closeUnderlyingByteChannel(self); err != nil {
		return fileIOError("error occurred while closing the channel. " + err.Error()), nil
	}
	return nil, nil
}

func initCharacterChannelModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	typCtx := semtypes.ContextFrom(env)
	jsonTy := semtypes.CreateJSON(typCtx)
	sld := semtypes.NewListDefinition()
	smd := semtypes.NewMappingDefinition()
	jmd := semtypes.NewMappingDefinition()
	jld := semtypes.NewListDefinition()
	types := characterChannelTypes{
		strArrTy:   sld.Define(env, nil, semtypes.ListRest(semtypes.String)),
		strMapTy:   smd.Define(env, nil, semtypes.String),
		jsonMapTy:  jmd.Define(env, nil, jsonTy),
		jsonListTy: jld.Define(env, nil, semtypes.ListRest(jsonTy)),
	}
	types.strMapAtom = semtypes.ToMappingAtomicType(typCtx, types.strMapTy)

	streamCompletionTy := semtypes.Union(semtypes.Error, semtypes.Nil)
	lsd := semtypes.NewStreamDefinition()
	types.lineRecordTy = closedNextRecordType(env, semtypes.String)
	types.lineRecordAtom = semtypes.ToMappingAtomicType(typCtx, types.lineRecordTy)
	types.lineStreamTy = lsd.Define(env, semtypes.String, streamCompletionTy)

	e := &characterChannelExterns{rt: rt, types: types}
	registerReadableCharacterChannelExterns(rt, e)
	registerWritableCharacterChannelExterns(rt, e)
}

func registerReadableCharacterChannelExterns(rt *runtime.Runtime, e *characterChannelExterns) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.initChannel", e.readableInitChannel)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.read", e.readableRead)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readString", e.readableReadString)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readAllLines", e.readableReadAllLines)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readJson", e.readableReadJson)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readXml", e.readableReadXml)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readProperty", e.readableReadProperty)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readAllProperties", e.readableReadAllProperties)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.lineStream", e.readableLineStream)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.close", e.readableClose)
}

func registerWritableCharacterChannelExterns(rt *runtime.Runtime, e *characterChannelExterns) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.initChannel", e.writableInitChannel)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.write", e.writableWrite)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.writeJson", e.writableWriteJson)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.writeXmlExtern", e.writableWriteXmlExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.writeProperties", e.writableWriteProperties)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.close", e.writableClose)
}

func init() {
	runtime.RegisterModuleInitializer(initCharacterChannelModule)
}
