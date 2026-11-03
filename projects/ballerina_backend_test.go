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

package projects_test

import (
	"path/filepath"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/test_util"
)

// loadBackendTestProject loads a single-file project with the given GenerateCode
// setting and returns its backend.
func loadBackendTestProject(t *testing.T, sourceDir string, generateCode bool) *projects.BallerinaBackend {
	t.Helper()
	require := test_util.NewRequire(t)

	absPath, err := filepath.Abs(filepath.Join(sourceDir, "main.bal"))
	require.NoError(err)

	buildOptions := projects.NewBuildOptionsBuilder().WithGenerateCode(generateCode).Build()
	result, err := loadProject(absPath, projects.ProjectLoadConfig{BuildOptions: &buildOptions})
	require.NoError(err)

	compilation := result.Project().CurrentPackage().Compilation()
	return projects.NewBallerinaBackend(compilation)
}

// TestBallerinaBackend_EagerGenerateCode_Succeeds covers GenerateCode(true) (the
// default): BIR is generated during Compilation(), and the backend reports no
// diagnostics and a non-nil BIR package.
func TestBallerinaBackend_EagerGenerateCode_Succeeds(t *testing.T) {
	t.Parallel()
	assert := test_util.New(t)

	backend := loadBackendTestProject(t, filepath.Join("testdata", "single-file"), true)

	assert.False(backend.DiagnosticResult().HasErrors())
	assert.NotNil(backend.BIR())
}

// TestBallerinaBackend_DeferredGenerateCode_Succeeds covers WithGenerateCode(false)
// followed by backend construction backfilling BIR generation successfully.
func TestBallerinaBackend_DeferredGenerateCode_Succeeds(t *testing.T) {
	t.Parallel()
	assert := test_util.New(t)

	backend := loadBackendTestProject(t, filepath.Join("testdata", "single-file"), false)

	assert.False(backend.DiagnosticResult().HasErrors())
	assert.NotNil(backend.BIR())
}

// TestBallerinaBackend_EagerGenerateCode_FailureReachesBackend guards the backend
// diagnostic contract: even when BIR generation already failed eagerly during
// Compilation() (GenerateCode(true), the default), building a backend from that
// same compilation must still surface the failure's diagnostics, not an empty
// result with a nil BIR package.
func TestBallerinaBackend_EagerGenerateCode_FailureReachesBackend(t *testing.T) {
	t.Parallel()
	assert := test_util.New(t)

	backend := loadBackendTestProject(t, filepath.Join("testdata", "single-file-birgen-fails"), true)

	assert.True(backend.DiagnosticResult().HasErrors())
	assert.True(backend.BIR() == nil)
}

// TestBallerinaBackend_DeferredGenerateCode_FailureReachesBackend covers the same
// failure with GenerateCode(false): backend construction runs BIR generation for
// the first time and must surface its diagnostics.
func TestBallerinaBackend_DeferredGenerateCode_FailureReachesBackend(t *testing.T) {
	t.Parallel()
	assert := test_util.New(t)

	backend := loadBackendTestProject(t, filepath.Join("testdata", "single-file-birgen-fails"), false)

	assert.True(backend.DiagnosticResult().HasErrors())
	assert.True(backend.BIR() == nil)
}

// TestBallerinaBackend_RepeatedAccessPreservesOutcome checks that a second
// NewBallerinaBackend call on the same compilation returns the cached backend
// instance rather than re-running (and potentially duplicating diagnostics for)
// BIR generation.
func TestBallerinaBackend_RepeatedAccessPreservesOutcome(t *testing.T) {
	t.Parallel()
	assert := test_util.New(t)
	require := test_util.NewRequire(t)

	absPath, err := filepath.Abs(filepath.Join("testdata", "single-file-birgen-fails", "main.bal"))
	require.NoError(err)

	result, err := loadProject(absPath)
	require.NoError(err)
	compilation := result.Project().CurrentPackage().Compilation()

	first := projects.NewBallerinaBackend(compilation)
	second := projects.NewBallerinaBackend(compilation)

	assert.Same(first, second)
	assert.True(first.DiagnosticResult().HasErrors())
	assert.Equal(len(first.DiagnosticResult().Diagnostics()), len(second.DiagnosticResult().Diagnostics()))
}
