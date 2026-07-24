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

import "ballerina/common"

type fixedLengthArray struct {
	initial     []SemType
	FixedLength int
}

func newFixedLengthArrayFromInitialFixedLength(initial []SemType, fixedLength int) fixedLengthArray {
	this := fixedLengthArray{}
	copiedInitial := make([]SemType, len(initial))
	copy(copiedInitial, initial)
	common.Assert(func() bool { return fixedLength >= 0 })
	this.initial = copiedInitial
	this.FixedLength = fixedLength
	return this
}

func fixedLengthArrayFrom(initial []SemType, fixedLength int) fixedLengthArray {
	return newFixedLengthArrayFromInitialFixedLength(initial, fixedLength)
}

func fixedLengthArrayEmpty() fixedLengthArray {
	return fixedLengthArrayFrom(nil, 0)
}
