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

package exec

import "ballerina/bir"

type callStackEntry struct {
	frame    *Frame
	location bir.Location
}

type callStack struct {
	elements []callStackEntry
}

func (cs *callStack) Push(frame *Frame) {
	cs.elements = append(cs.elements, callStackEntry{frame: frame})
}

func (cs *callStack) Pop() {
	cs.elements = cs.elements[:len(cs.elements)-1]
}

func (cs *callStack) top() *Frame {
	return cs.elements[len(cs.elements)-1].frame
}

func (cs *callStack) len() int {
	return len(cs.elements)
}

func (cs *callStack) SetCurrentLocation(location bir.Location) {
	if len(cs.elements) == 0 {
		return
	}
	cs.elements[len(cs.elements)-1].location = location
}

// Entries returns the current entries in the call stack from bottom to top.
func (cs *callStack) Entries() []callStackEntry {
	entries := make([]callStackEntry, len(cs.elements))
	copy(entries, cs.elements)
	return entries
}
