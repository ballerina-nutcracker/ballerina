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

//go:build linux

package executable

import (
	"debug/elf"
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// linuxSectionName mirrors splice.ELFSectionName — duplicated so balrt
// never pulls in the packer package it doesn't need (only bal build packs).
const linuxSectionName = ".balexe.payload"

// tryLoadFrom looks for linuxSectionName in exe, a 64-bit ELF binary. A
// missing section and a parse failure both mean "not a bal-packed
// binary" — (nil, nil, nil), not an error; there is no other format to
// fall back to on linux.
func tryLoadFrom(exe string) ([]*bir.BIRPackage, semtypes.Env, error) {
	f, err := elf.Open(exe)
	if err != nil {
		return nil, nil, nil
	}
	defer func() { _ = f.Close() }()

	sec := f.Section(linuxSectionName)
	if sec == nil {
		return nil, nil, nil
	}
	payload, err := sec.Data()
	if err != nil {
		return nil, nil, fmt.Errorf("reading embedded payload section: %w", err)
	}

	pkgs, tyEnv, err := unmarshalPayload(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("corrupt embedded program: %w", err)
	}
	return pkgs, tyEnv, nil
}
