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

package main

import (
	"fmt"
	"os"

	"ballerina-lang-go/cli/internal/executable"
	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/runtime"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "bal",
	Short:         "The build system and package manager of Ballerina",
	Long:          `The build system and package manager of Ballerina`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	// Check whether this binary is a compiled Ballerina program (produced by
	// bal build). If so, run the embedded BIR directly and skip the CLI.
	birPkgs, tyEnv, err := executable.TryLoad()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ballerina:", err)
		os.Exit(1)
	}
	if birPkgs != nil {
		pal, cleanupSignals := palnative.NewPlatform()
		defer cleanupSignals()
		rt := runtime.NewRuntime(pal, tyEnv)
		var initErr error
		for _, pkg := range birPkgs {
			if err := rt.Init(*pkg); err != nil {
				fmt.Fprintln(os.Stderr, err)
				initErr = err
				break
			}
		}
		rt.Listen()
		if initErr != nil {
			os.Exit(1)
		}
		exitCode := <-rt.ExitStatus
		if exitCode != 0 {
			os.Exit(int(exitCode))
		}
		return
	}

	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(packCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
