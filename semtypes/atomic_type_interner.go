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

package semtypes

// AtomicTypeInterner assigns handles using atomic-type pointer identity. It is
// thread compatible; callers must synchronize concurrent access.
type AtomicTypeInterner struct {
	handles map[AtomicType]InternHandle
}

func NewAtomicTypeInterner() *AtomicTypeInterner {
	return &AtomicTypeInterner{handles: make(map[AtomicType]InternHandle)}
}

func (i *AtomicTypeInterner) Intern(atom AtomicType) InternHandle {
	if handle, ok := i.handles[atom]; ok {
		return handle
	}
	handle := InternHandle(len(i.handles))
	i.handles[atom] = handle
	return handle
}

func (i *AtomicTypeInterner) Lookup(atom AtomicType) (InternHandle, bool) {
	handle, ok := i.handles[atom]
	return handle, ok
}
