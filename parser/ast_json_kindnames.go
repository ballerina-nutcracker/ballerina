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

package parser

import (
	"fmt"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/st"
)

// kindName returns the generated constant name for a SyntaxKind (matching Java enum.name()).
func kindName(kind st.SyntaxKind) string {
	name := kind.String()
	if strings.HasPrefix(name, "SyntaxKind(") {
		return fmt.Sprintf("UNKNOWN_KIND_%d", kind)
	}
	return name
}
