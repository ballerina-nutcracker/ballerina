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

//go:build windows

package executable

import (
	"debug/pe"
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// windowsSectionName mirrors splice.PESectionName — duplicated so balrt
// never pulls in the packer package it doesn't need (only bal build packs).
const windowsSectionName = ".balexe"

// tryLoadFrom looks for windowsSectionName in exe, a 64-bit PE binary. A
// missing section and a parse failure both mean "not a bal-packed
// binary" — (nil, nil, nil), not an error.
func tryLoadFrom(exe string) ([]*bir.BIRPackage, semtypes.Env, error) {
	f, err := pe.Open(exe)
	if err != nil {
		return nil, nil, nil
	}
	defer func() { _ = f.Close() }()

	sec := f.Section(windowsSectionName)
	if sec == nil {
		return nil, nil, nil
	}
	raw, err := sec.Data()
	if err != nil {
		return nil, nil, fmt.Errorf("reading embedded payload section: %w", err)
	}
	// Data() returns FileAlignment-padded bytes; VirtualSize (set to the
	// true payload length by splicePESection) is the real boundary.
	if uint64(sec.VirtualSize) > uint64(len(raw)) {
		return nil, nil, fmt.Errorf("payload section VirtualSize %d exceeds raw data length %d", sec.VirtualSize, len(raw))
	}
	payload := raw[:sec.VirtualSize]

	pkgs, tyEnv, err := unmarshalPayload(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("corrupt embedded program: %w", err)
	}
	return pkgs, tyEnv, nil
}
