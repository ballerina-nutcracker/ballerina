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

package model

import (
	"strings"
	"sync"
)

type Name string

func (n *Name) Value() string {
	return string(*n)
}

const (
	DEFAULT_PACKAGE          = Name(".")
	IGNORE                   = Name("_")
	ANON_ORG                 = Name("$anon")
	NIL_VALUE                = Name("()")
	USER_DEFINED_INIT_SUFFIX = Name("init")
	DEFAULT_VERSION          = Name("0.0.0")
)

// You should never directly allocate a PackageID. Instead, use the NewPackageID function.
type PackageID struct {
	OrgName        *Name
	PkgName        *Name
	Name           *Name
	Version        *Name
	NameComps      []Name
	SourceFileName *Name
	SourceRoot     *string
	isUnnamed      bool
	SkipTests      bool
	IsTestPkg      bool
}

func (p *PackageID) IsUnnamed() bool {
	return p.isUnnamed || (p.OrgName == nil && p.PkgName == nil && p.Version == nil)
}

func NewPackageID(interner *PackageIDInterner, orgName Name, nameComps []Name, version Name) *PackageID {
	nameParts := make([]string, len(nameComps))
	for i, name := range nameComps {
		nameParts[i] = string(name)
	}
	name := strings.Join(nameParts, ".")
	id := &PackageID{
		OrgName:   &orgName,
		NameComps: nameComps,
		Name:      new(Name(name)),
		PkgName:   new(Name(name)),
		Version:   &version,
		SkipTests: true,
	}
	return interner.Intern(id)
}

var DefaultPackageIDInterner = &PackageIDInterner{
	packageMap: make(map[packageKey]*PackageID),
}

var (
	DEFAULT         = NewPackageID(DefaultPackageIDInterner, ANON_ORG, []Name{DEFAULT_PACKAGE}, DEFAULT_VERSION)
	ANNOTATIONS_PKG = NewPackageID(DefaultPackageIDInterner, Name("ballerina"), []Name{Name("lang"), Name("annotations")}, DEFAULT_VERSION)
	INTERNAL_PKG    = NewPackageID(DefaultPackageIDInterner, Name("ballerina"), []Name{Name("lang"), Name("__internal")}, DEFAULT_VERSION)
)

func CreateNameComps(name Name) []Name {
	if name == "." {
		return []Name{Name(".")}
	}
	parts := strings.Split(name.Value(), ".")
	result := make([]Name, len(parts))
	for i, part := range parts {
		result[i] = Name(part)
	}
	return result
}

type PackageIDInterner struct {
	rwLock     sync.RWMutex
	packageMap map[packageKey]*PackageID
}

func (p *PackageIDInterner) GetDefaultPackage() *PackageID {
	return DEFAULT
}

func (p *PackageIDInterner) Intern(packageID *PackageID) *PackageID {
	packageKey := packageKeyFromPackageID(packageID)
	p.rwLock.RLock()
	internedPackage, ok := p.packageMap[packageKey]
	p.rwLock.RUnlock()
	if ok {
		return internedPackage
	}
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	p.packageMap[packageKey] = packageID
	return packageID
}

type packageKey struct {
	orgName Name
	pkgName Name
	name    Name
	version Name
}

func packageKeyFromPackageID(packageID *PackageID) packageKey {
	if packageID == nil || packageID.IsUnnamed() {
		return packageKey{
			orgName: ANON_ORG,
			pkgName: DEFAULT_PACKAGE,
			version: DEFAULT_VERSION,
			name:    DEFAULT_PACKAGE,
		}
	}
	return packageKey{
		orgName: *packageID.OrgName,
		pkgName: *packageID.PkgName,
		version: *packageID.Version,
		name:    *packageID.Name,
	}
}
