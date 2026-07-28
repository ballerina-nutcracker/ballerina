// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package tomlparser

import (
	"fmt"

	"ballerina/tools/diagnostics"
)

type Validator interface {
	Validate(toml *Toml) error
}

type validatorImpl struct {
	schema Schema
}

func NewValidator(schema Schema) Validator {
	return &validatorImpl{
		schema: schema,
	}
}

func (v *validatorImpl) Validate(toml *Toml) error {
	if toml == nil {
		return fmt.Errorf("toml document is nil")
	}

	data := toml.ToMap()

	if err := v.schema.Validate(data); err != nil {
		diagnostic := Diagnostic{
			Message:  err.Error(),
			Severity: diagnostics.Error,
		}
		toml.diagnostics = append(toml.diagnostics, diagnostic)
		return err
	}

	return nil
}
