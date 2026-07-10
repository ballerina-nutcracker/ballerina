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

package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"ballerina/ast"
	"ballerina/model"
)

type CPEntryType uint8

const (
	CP_ENTRY_STRING CPEntryType = iota + 1
	CP_ENTRY_PACKAGE
	CP_ENTRY_SHAPE
)

type CPEntry interface {
	EntryType() CPEntryType
}

type (
	StringCPEntry struct {
		Value string
	}
	PackageCPEntry struct {
		OrgNameCPIndex    int32
		PkgNameCPIndex    int32
		ModuleNameCPIndex int32
		VersionCPIndex    int32
	}
	ShapeCPEntry struct {
		Shape ast.BType
	}
)

func (e *StringCPEntry) EntryType() CPEntryType {
	return CP_ENTRY_STRING
}

func (e *PackageCPEntry) EntryType() CPEntryType {
	return CP_ENTRY_PACKAGE
}

func (e *ShapeCPEntry) EntryType() CPEntryType {
	return CP_ENTRY_SHAPE
}

type ConstantPool struct {
	entries  []CPEntry
	entryMap map[string]int
}

func NewConstantPool() *ConstantPool {
	return &ConstantPool{
		entries:  make([]CPEntry, 0),
		entryMap: make(map[string]int),
	}
}

func (cp *ConstantPool) EntryKey(entry CPEntry) string {
	switch e := entry.(type) {
	case *StringCPEntry:
		return fmt.Sprintf("str:%s", e.Value)
	case *PackageCPEntry:
		return fmt.Sprintf("pkg:%d:%d:%d:%d", e.OrgNameCPIndex, e.PkgNameCPIndex, e.ModuleNameCPIndex, e.VersionCPIndex)
	case *ShapeCPEntry:
		panic("shape key generation not implemented")
	default:
		panic("unknown CPEntry type")
	}
}

func (cp *ConstantPool) AddEntry(entry CPEntry) int32 {
	key := cp.EntryKey(entry)
	if index, exists := cp.entryMap[key]; exists {
		return int32(index)
	}

	index := len(cp.entries)
	cp.entries = append(cp.entries, entry)
	cp.entryMap[key] = index
	return int32(index)
}

func (cp *ConstantPool) AddStringCPEntry(value string) int32 {
	return cp.AddEntry(&StringCPEntry{Value: value})
}

func (cp *ConstantPool) AddPackageCPEntry(pkg *model.PackageID) int32 {
	return cp.AddEntry(&PackageCPEntry{
		OrgNameCPIndex:    cp.AddStringCPEntry(pkg.OrgName.Value()),
		PkgNameCPIndex:    cp.AddStringCPEntry(pkg.PkgName.Value()),
		ModuleNameCPIndex: cp.AddStringCPEntry(pkg.Name.Value()),
		VersionCPIndex:    cp.AddStringCPEntry(pkg.Version.Value()),
	})
}

func (cp *ConstantPool) AddShapeCPEntry(shape ast.BType) int32 {
	panic("shape entry addition not implemented")
}

func (cp *ConstantPool) Serialize() ([]byte, error) {
	var errMsg string
	defer func() {
		if r := recover(); r != nil {
			errMsg = fmt.Sprintf("%v", r)
		}
	}()

	buf := &bytes.Buffer{}

	write(buf, int64(-1)) // entry count placeholder

	for _, entry := range cp.entries {
		cp.WriteCPEntry(buf, entry)
	}

	bytes := buf.Bytes()
	binary.BigEndian.PutUint64(bytes[0:8], uint64(len(cp.entries)))

	if errMsg != "" {
		return nil, fmt.Errorf("constant pool serialization failed due to %s", errMsg)
	}

	return bytes, nil
}

func (cp *ConstantPool) WriteCPEntry(buf *bytes.Buffer, entry CPEntry) {
	entryType := entry.EntryType()
	write(buf, int8(entryType))

	switch e := entry.(type) {
	case *StringCPEntry:
		strBytes := []byte(e.Value)
		cp.writeLength(buf, len(strBytes))
		_, err := buf.Write(strBytes)
		if err != nil {
			panic(fmt.Sprintf("writing string bytes: %v", err))
		}
	case *PackageCPEntry:
		write(buf, e.OrgNameCPIndex)
		write(buf, e.PkgNameCPIndex)
		write(buf, e.ModuleNameCPIndex)
		write(buf, e.VersionCPIndex)
	case *ShapeCPEntry:
		panic("shape serialization not implemented")
	default:
		panic(fmt.Sprintf("unsupported constant pool entry type: %T", entry))
	}
}

func (cp *ConstantPool) writeLength(buf *bytes.Buffer, length int) {
	write(buf, int64(length))
}

func write(buf *bytes.Buffer, data any) {
	if err := binary.Write(buf, binary.BigEndian, data); err != nil {
		panic(fmt.Sprintf("writing binary data: %v", err))
	}
}
