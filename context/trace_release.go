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

//go:build !debug

package context

import "github.com/ballerina-nutcracker/ballerina/model"

// TraceSpan is an empty handle in normal builds; every operation on it is a
// no-op the compiler can eliminate.
type TraceSpan struct{}

// traceState is empty in normal builds: no clocks, locks, or span storage.
type traceState struct{}

func newTraceState(_ TraceOptions) traceState {
	return traceState{}
}

// StartNamedSpan records nothing in normal builds.
func (c *CompilerContext) StartNamedSpan(_, _ string) TraceSpan {
	return TraceSpan{}
}

// StartPackageSpan records nothing in normal builds.
func (c *CompilerContext) StartPackageSpan(_ string, _ *model.PackageID) TraceSpan {
	return TraceSpan{}
}

// StartChild records nothing in normal builds.
func (s TraceSpan) StartChild(_, _ string) TraceSpan {
	return TraceSpan{}
}

// End records nothing in normal builds.
func (s TraceSpan) End() {}

// TraceJSON returns no recording in normal builds.
func (c *CompilerEnvironment) TraceJSON() ([]byte, error) {
	return nil, nil
}
