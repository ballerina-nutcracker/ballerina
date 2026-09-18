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

package main

import (
	"github.com/ballerina-nutcracker/ballerina/context"

	"github.com/spf13/cobra"
)

// runTraceOutput does no trace work in normal builds.
type runTraceOutput struct{}

func newRunTraceOutput(_ string) *runTraceOutput { return &runTraceOutput{} }

func registerTraceFlag(_ *cobra.Command) {}

func runTraceOptions(_ *cobra.Command) (options context.TraceOptions, path string, err error) {
	return context.TraceOptions{}, "", nil
}

func (o *runTraceOutput) finalize(_ *context.CompilerEnvironment) error { return nil }

// reportFailureDuringPanic leaves the normal build's panic path untouched.
func (o *runTraceOutput) reportFailureDuringPanic(_ *context.CompilerEnvironment) {}
