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

package corpus

import (
	"os"
	"testing"

	"ballerina/context"
	"ballerina/projects"
	"ballerina/semtypes"
	"ballerina/test_util/testphases"
)

func BenchmarkLanguageService(b *testing.B) {
	for _, tc := range benchTestPairs(b) {
		if tc.IsProject {
			benchmarkLanguageServiceProject(b, tc.Name, tc.InputPath)
			continue
		}
		benchmarkLanguageServiceFile(b, tc.Name, tc.InputPath)
	}
}

func benchmarkLanguageServiceFile(b *testing.B, name, inputPath string) {
	contentBytes, err := os.ReadFile(inputPath)
	if err != nil {
		b.Fatalf("read %s: %v", inputPath, err)
	}
	content := string(contentBytes)

	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
			cx := context.NewCompilerContext(env)
			_, err := testphases.RunPipelineWithContent(env, cx, nil, testphases.PhaseDesugar, inputPath, content)
			if err != nil {
				b.Fatalf("language service benchmark failed for %s: %v", inputPath, err)
			}
			if cx.HasDiagnostics() {
				b.Fatalf("language service benchmark produced diagnostics for %s", inputPath)
			}
		}
	})
}

func benchmarkLanguageServiceProject(b *testing.B, name, inputPath string) {
	ballerinaEnvPath, err := getBallerinaEnvPath()
	if err != nil {
		b.Fatalf("getBallerinaEnvPath: %v", err)
	}
	ballerinaEnvFs := os.DirFS(ballerinaEnvPath)

	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			result, err := projects.Load(os.DirFS(inputPath), ".", projects.ProjectLoadConfig{
				BallerinaEnvFs: ballerinaEnvFs,
			})
			if err != nil {
				b.Fatalf("language service benchmark failed to load %s: %v", inputPath, err)
			}
			compilation := result.Project().CurrentPackage().Compilation()
			if diagnostics := compilation.DiagnosticResult(); diagnostics.DiagnosticCount() > 0 {
				b.Fatalf("language service benchmark produced diagnostics for %s", inputPath)
			}
		}
	})
}
