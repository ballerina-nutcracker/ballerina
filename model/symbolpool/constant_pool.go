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

// Package symbolpool provide logic necessary to serialize [ExportedSymbolSpace] to binary
package symbolpool

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type constantPool struct {
	entries  []string
	entryMap map[string]int32
}

func newConstantPool() *constantPool {
	return &constantPool{
		entries:  make([]string, 0),
		entryMap: make(map[string]int32),
	}
}

func (cp *constantPool) addString(value string) int32 {
	if idx, exists := cp.entryMap[value]; exists {
		return idx
	}
	idx := int32(len(cp.entries))
	cp.entries = append(cp.entries, value)
	cp.entryMap[value] = idx
	return idx
}

func (cp *constantPool) serialize() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, int64(len(cp.entries))); err != nil {
		return nil, err
	}
	for _, entry := range cp.entries {
		strBytes := []byte(entry)
		if err := binary.Write(buf, binary.BigEndian, int64(len(strBytes))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(strBytes); err != nil {
			return nil, fmt.Errorf("writing string bytes: %v", err)
		}
	}
	return buf.Bytes(), nil
}

func deserializeConstantPool(r *bytes.Reader) []string {
	var count int64
	read(r, &count)
	entries := make([]string, count)
	for i := int64(0); i < count; i++ {
		var length int64
		read(r, &length)
		strBytes := make([]byte, length)
		_, err := r.Read(strBytes)
		if err != nil {
			panic(fmt.Sprintf("reading string bytes: %v", err))
		}
		entries[i] = string(strBytes)
	}
	return entries
}

func read(r *bytes.Reader, v any) {
	if err := binary.Read(r, binary.BigEndian, v); err != nil {
		panic(fmt.Sprintf("reading binary data: %v", err))
	}
}
