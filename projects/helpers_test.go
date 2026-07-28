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
	"os"
	"path/filepath"

	"ballerina/projects"
)

// loadProject loads a Ballerina project for tests. It always forces
// Offline=true on the BuildOptions so tests never make network calls —
// repositories behave purely as on-disk caches.
//
// Caller-supplied BuildOptions, Repositories and BallerinaEnvFs flow through
// projects.ProjectLoadConfig as before; only Offline is unconditionally
// pinned by overlaying it via BuildOptions.AcceptTheirs.
func loadProject(path string, config ...projects.ProjectLoadConfig) (projects.ProjectLoadResult, error) {
	baseDir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		baseDir = filepath.Dir(path)
		path = filepath.Base(path)
	} else {
		path = "."
	}

	fsys := os.DirFS(baseDir)

	var cfg projects.ProjectLoadConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	if cfg.BallerinaEnvFs == nil {
		ballerinaEnvPath, err := getBallerinaEnvPath()
		if err != nil {
			return projects.ProjectLoadResult{}, err
		}
		cfg.BallerinaEnvFs = os.DirFS(ballerinaEnvPath)
	}

	// Force Offline=true on whatever BuildOptions the caller provided.
	var base projects.BuildOptions
	if cfg.BuildOptions != nil {
		base = *cfg.BuildOptions
	} else {
		base = projects.NewBuildOptions()
	}
	offlineOverlay := projects.NewBuildOptionsBuilder().WithOffline(true).Build()
	merged := base.AcceptTheirs(offlineOverlay)
	cfg.BuildOptions = &merged

	return projects.Load(fsys, path, cfg)
}

func getBallerinaEnvPath() (string, error) {
	if balEnv := os.Getenv(projects.BallerinaEnvVar); balEnv != "" {
		return balEnv, nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(userHome, projects.UserHomeDirName), nil
}
