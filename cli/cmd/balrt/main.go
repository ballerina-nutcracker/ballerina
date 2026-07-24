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

// Command balrt is a slim runtime-only Ballerina executable: it embeds the
// interpreter and all lib/rt native packages, but none of the compiler,
// parser, semantic analysis, or CLI code that "bal" carries. It has no
// standalone purpose — it only exists to be the stub that "bal build" embeds
// a compiled program's BIR payload into (see
// cli/internal/executable.ResolveStub), producing a smaller artifact than
// using the full "bal" binary as the stub.
package main

import (
	"fmt"
	"os"

	"ballerina-lang-go/cli/internal/executable"
	_ "ballerina-lang-go/lib/rt"
)

func main() {
	birPkgs, tyEnv, err := executable.TryLoad()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ballerina:", err)
		os.Exit(1)
	}
	if birPkgs == nil {
		fmt.Fprintln(os.Stderr, "ballerina: balrt only runs compiled Ballerina executables produced by bal build")
		os.Exit(1)
	}
	os.Exit(executable.Run(birPkgs, tyEnv))
}
