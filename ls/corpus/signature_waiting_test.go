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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package corpus

import (
	"context"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/projects"
)

const awaitSignatureDependency = "$pal/awaitSignatureDependency"
const releaseSignatureDependency = "$pal/releaseSignatureDependency"

// signatureGateRepository holds actual background dependency resolution before
// a generation can seal. It uses the existing repository injection seam only.
type signatureGateRepository struct {
	projects.Repository
	entered     chan struct{}
	released    chan struct{}
	enterOnce   sync.Once
	releaseOnce sync.Once
}

func newSignatureGateRepository(repository projects.Repository) *signatureGateRepository {
	return &signatureGateRepository{Repository: repository, entered: make(chan struct{}), released: make(chan struct{})}
}

func (r *signatureGateRepository) await(ctx context.Context) error {
	r.enterOnce.Do(func() { close(r.entered) })
	select {
	case <-r.released:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *signatureGateRepository) release() { r.releaseOnce.Do(func() { close(r.released) }) }

func (r *signatureGateRepository) GetPackageVersions(ctx context.Context, org, name string, options projects.ResolutionOptions) ([]projects.PackageVersion, error) {
	if err := r.await(ctx); err != nil {
		return nil, err
	}
	return r.Repository.GetPackageVersions(ctx, org, name, options)
}

func (r *signatureGateRepository) GetPackage(ctx context.Context, org, name, version string, options projects.ResolutionOptions) (*projects.Package, error) {
	if err := r.await(ctx); err != nil {
		return nil, err
	}
	return r.Repository.GetPackage(ctx, org, name, version, options)
}
