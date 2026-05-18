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
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the root of the bal CLI. SetErrPrefix gives every error printed
// by cobra (both flag-parse errors and errors returned from RunE) the
// "ballerina:" prefix expected by users. SilenceUsage stays true because
// subcommands embed their own concise USAGE block in the error (see
// usageError); cobra's verbose UsageString would otherwise duplicate that.
var rootCmd = &cobra.Command{
	Use:          "bal",
	Short:        "The build system and package manager of Ballerina",
	Long:         `The build system and package manager of Ballerina`,
	SilenceUsage: true,
}

func main() {
	rootCmd.SetErrPrefix("ballerina:")

	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(packCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
