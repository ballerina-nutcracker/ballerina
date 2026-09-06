// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package projects

import (
	"context"
	"testing"
	"testing/fstest"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type recordingPackageResolver struct {
	packageOptions ResolutionOptions
	nameOptions    ResolutionOptions
	request        ResolutionRequest
	remoteCalls    int
}

func (r *recordingPackageResolver) ResolvePackages(_ context.Context, requests []ResolutionRequest,
	options ResolutionOptions,
) []ResolutionResponse {
	r.packageOptions = options
	r.request = requests[0]
	if !options.Offline() {
		r.remoteCalls++
	}
	return []ResolutionResponse{NewUnresolvedResponse(requests[0])}
}

func (r *recordingPackageResolver) ResolveByName(_ context.Context, _, _ string, options ResolutionOptions) []*Package {
	r.nameOptions = options
	if !options.Offline() {
		r.remoteCalls++
	}
	return nil
}

func (*recordingPackageResolver) AddRepository(Repository)   {}
func (*recordingPackageResolver) Repositories() []Repository { return nil }

func TestDriverResolverAppliesOfflineAndStickyToConcreteLookups(t *testing.T) {
	recording := &recordingPackageResolver{}
	environment := NewEnvironment(fstest.MapFS{}, compilercontext.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false))
	environment.packageResolver = recording
	resolver := &DriverDependencyResolver{environment: environment, native: make(map[driver.PackageDescriptor]*BalaProject)}
	resolution := driver.ResolutionOptions{Offline: true, Sticky: true}
	locked := driver.PackageDescriptor{Org: "acme", Name: "locked", Version: "2.3.4"}
	if _, _, _, _, err := resolver.Resolve(context.Background(), driver.DependencyRequest{Descriptor: locked, Resolution: resolution}); err != nil {
		t.Fatal(err)
	}
	if !recording.packageOptions.Offline() || !recording.packageOptions.Sticky() {
		t.Fatalf("versioned lookup options = %+v, want offline+sticky", recording.packageOptions)
	}
	if got := recording.request.Descriptor().Version().String(); got != locked.Version {
		t.Fatalf("sticky lookup version = %q, want locked %q", got, locked.Version)
	}
	if _, _, _, _, err := resolver.Resolve(context.Background(), driver.DependencyRequest{
		Descriptor: driver.PackageDescriptor{Org: "acme", Name: "versionless"}, Resolution: resolution,
	}); err != nil {
		t.Fatal(err)
	}
	if !recording.nameOptions.Offline() || !recording.nameOptions.Sticky() {
		t.Fatalf("versionless lookup options = %+v, want offline+sticky", recording.nameOptions)
	}
	if recording.remoteCalls != 0 {
		t.Fatalf("remote lookups = %d, want 0 in offline mode", recording.remoteCalls)
	}
}
