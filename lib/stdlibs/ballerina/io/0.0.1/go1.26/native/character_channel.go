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
	"strings"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"

	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
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

func charReaderOf(self *values.Object) (*bufio.Reader, bool) {
	v, ok := self.Get("$charReader")
	if !ok {
		return nil, false
	}
	r, ok := v.(*bufio.Reader)
	return r, ok
}

func byteChannelOf(self *values.Object) (*values.Object, bool) {
	v, ok := self.Get("$byteChannel")
	if !ok {
		return nil, false
	}
	obj, ok := v.(*values.Object)
	return obj, ok
}

func charsetOf(self *values.Object) (encoding.Encoding, bool) {
	v, ok := self.Get("$charset")
	if !ok {
		return nil, false
	}
	enc, ok := v.(encoding.Encoding)
	return enc, ok
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
		return closer.Close()
	}
	if writer, ok := writerOf(byteCh); ok {
		return writer.Close()
	}
	return nil
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
				var code rune
				if _, err := fmt.Sscanf(s[i+1:i+5], "%04x", &code); err == nil {
					sb.WriteRune(code)
					i += 4
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
			if r < 0x20 || r > 0x7e {
				fmt.Fprintf(&sb, `\u%04X`, r)
			} else {
				sb.WriteRune(r)
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

func initCharacterChannelModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	typCtx := semtypes.ContextFrom(env)
	jsonTy := semtypes.CreateJSON(typCtx)
	sld := semtypes.NewListDefinition()
	smd := semtypes.NewMappingDefinition()
	jmd := semtypes.NewMappingDefinition()
	jld := semtypes.NewListDefinition()
	types := characterChannelTypes{
		strArrTy:   sld.DefineListTypeWrappedWithEnvSemType(env, semtypes.STRING),
		strMapTy:   smd.DefineMappingTypeWrapped(env, nil, semtypes.STRING),
		jsonMapTy:  jmd.DefineMappingTypeWrapped(env, nil, jsonTy),
		jsonListTy: jld.DefineListTypeWrappedWithEnvSemType(env, jsonTy),
	}
	types.strMapAtom = semtypes.ToMappingAtomicType(typCtx, types.strMapTy)

	streamCompletionTy := semtypes.Union(semtypes.ERROR, semtypes.NIL)
	lsd := semtypes.NewStreamDefinition()
	types.lineRecordTy = closedNextRecordType(env, semtypes.STRING)
	types.lineRecordAtom = semtypes.ToMappingAtomicType(typCtx, types.lineRecordTy)
	types.lineStreamTy = lsd.Define(env, semtypes.STRING, streamCompletionTy)

	registerReadableCharacterChannelExterns(rt, types)
	registerWritableCharacterChannelExterns(rt, types)
}

func registerReadableCharacterChannelExterns(rt *runtime.Runtime, types characterChannelTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.initChannel",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.read",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			numberOfChars, _ := args[1].(int64)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			if eofReached(self) {
				return byteChannelEofError(), nil
			}
			r, _ := charReaderOf(self)
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
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readString",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			r, _ := charReaderOf(self)
			content, err := drainChars(self, r)
			if err != nil {
				return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
			}
			return strings.Join(splitLines([]byte(content)), "\n"), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readAllLines",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			r, _ := charReaderOf(self)
			content, err := drainChars(self, r)
			if err != nil {
				return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
			}
			lines := splitLines([]byte(content))
			items := make([]values.BalValue, len(lines))
			for i, line := range lines {
				items[i] = line
			}
			return values.NewList(types.strArrTy, semtypes.ToListAtomicType(semtypes.ContextFrom(rt.GetTypeEnv()), types.strArrTy), false, nil, 0, items), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readJson",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			r, _ := charReaderOf(self)
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
			return values.GoToBalValue(ctx.TypeCtx, raw, types.jsonListTy, types.jsonMapTy), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readXml",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			r, _ := charReaderOf(self)
			content, err := drainChars(self, r)
			if err != nil {
				return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
			}
			xmlVal, parseErr := values.ParseAsXMLValue(ctx.TypeCtx, values.FromBytes([]byte(content)), values.XMLLenientMode)
			if parseErr != nil {
				return fileIOError("error occurred while reading xml from the channel. " + parseErr.Error()), nil
			}
			return xmlVal, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readProperty",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			key, _ := args[1].(string)
			defaultValue, _ := args[2].(string)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			r, _ := charReaderOf(self)
			props, err := channelProperties(self, r)
			if err != nil {
				return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
			}
			for _, entry := range props {
				if entry.key == key {
					return entry.value, nil
				}
			}
			return defaultValue, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.readAllProperties",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			r, _ := charReaderOf(self)
			props, err := channelProperties(self, r)
			if err != nil {
				return fileIOError("error occurred while reading characters from the channel. " + err.Error()), nil
			}
			entries := make([]values.MapEntry, len(props))
			for i, entry := range props {
				entries[i] = values.MapEntry{Key: entry.key, Value: entry.value}
			}
			return values.NewMap(types.strMapTy, types.strMapAtom, false, entries), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.lineStream",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			r, _ := charReaderOf(self)
			next := func() values.BalValue {
				line, ok, readErr := readLineCRLF(r)
				if readErr != nil {
					return fileIOError("error occurred while reading characters from the channel. " + readErr.Error())
				}
				if !ok {
					self.Put("$eof", true)
					_ = closeUnderlyingByteChannel(self)
					return nil
				}
				return values.NewMap(types.lineRecordTy, types.lineRecordAtom, false,
					[]values.MapEntry{{Key: "value", Value: line}})
			}
			closeFn := func() values.BalValue {
				if err := closeUnderlyingByteChannel(self); err != nil {
					return fileIOError("error occurred while closing the channel. " + err.Error())
				}
				return nil
			}
			return values.NewStream(types.lineStreamTy, next, closeFn), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "ReadableCharacterChannel.close",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			markClosed(self)
			if err := closeUnderlyingByteChannel(self); err != nil {
				return fileIOError("error occurred while closing the channel. " + err.Error()), nil
			}
			return nil, nil
		})
}

func registerWritableCharacterChannelExterns(rt *runtime.Runtime, types characterChannelTypes) {
	writeEncoded := func(self *values.Object, content string) values.BalValue {
		enc, _ := charsetOf(self)
		byteCh, _ := byteChannelOf(self)
		writer, ok := writerOf(byteCh)
		if !ok {
			return fileIOError("Byte channel is not initialized")
		}
		data, err := enc.NewEncoder().Bytes([]byte(content))
		if err != nil {
			return fileIOError("error occurred while writing characters to the channel. " + err.Error())
		}
		if _, err := writer.Write(data); err != nil {
			return fileIOError("error occurred while writing characters to the channel. " + err.Error())
		}
		return nil
	}

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.initChannel",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			byteCh, _ := args[1].(*values.Object)
			charset, _ := args[2].(string)
			enc, err := lookupCharset(charset)
			if err != nil {
				// A Go error return panics in the interpreter, matching
				// jBallerina, where init throws for an unsupported charset.
				return nil, err
			}
			self.Put("$charset", enc)
			self.Put("$byteChannel", byteCh)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.write",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
			enc, _ := charsetOf(self)
			byteCh, _ := byteChannelOf(self)
			writer, ok := writerOf(byteCh)
			if !ok {
				return fileIOError("Byte channel is not initialized"), nil
			}
			data, err := enc.NewEncoder().Bytes([]byte(string(runes[startOffset:])))
			if err != nil {
				return fileIOError("error occurred while writing characters to the channel. " + err.Error()), nil
			}
			if _, err := writer.Write(data); err != nil {
				return fileIOError("error occurred while writing characters to the channel. " + err.Error()), nil
			}
			return int64(len(data)), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.writeJson",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			data, err := values.ToJSONByteArray(args[1])
			if err != nil {
				return fileIOError("error occurred while serializing json. " + err.Error()), nil
			}
			return writeEncoded(self, string(data)), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.writeXmlExtern",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
			return writeEncoded(self, writeContent), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.writeProperties",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
			sb.WriteString("#" + comment + "\n")
			sb.WriteString("#" + time.Now().Format("Mon Jan 2 15:04:05 MST 2006") + "\n")
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
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "WritableCharacterChannel.close",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			if isClosed(self) {
				return charChannelClosedError(), nil
			}
			markClosed(self)
			if err := closeUnderlyingByteChannel(self); err != nil {
				return fileIOError("error occurred while closing the channel. " + err.Error()), nil
			}
			return nil, nil
		})
}

func init() {
	runtime.RegisterModuleInitializer(initCharacterChannelModule)
}
