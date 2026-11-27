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

package native

import (
	"errors"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// TestReceiveFailedConversion checks that a message whose conversion fails is not marked as
// received, so the next read returns it again.
// Conversion cannot fail from Ballerina, so the test replaces it.
func TestReceiveFailedConversion(t *testing.T) {
	saved := convertMessage
	// Not parallel: overrides the package-level convertMessage.
	convertMessage = func(*extern.Context, semtypes.SemType, []byte) (values.BalValue, error) {
		return nil, errors.New("conversion failed")
	}
	t.Cleanup(func() { convertMessage = saved })
	var marked []bool
	retrieve := func() ([]byte, func(bool), *values.Error) {
		return []byte("Subject: x\r\n\r\nx\r\n"), func(received bool) { marked = append(marked, received) }, nil
	}
	platform := pal.Platform{Time: pal.Time{MonotonicNow: func() time.Duration { return 0 }}}
	_, err := (&mailbox{}).receive(nil, platform, semtypes.SemType{}, decimal.FromInt64(0), retrieve)
	if err == nil || len(marked) != 1 || marked[0] {
		t.Errorf("receive error = %v, finish calls = %v; want the error and the message left unread", err, marked)
	}
}
