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
	"testing"

	"golang.org/x/text/transform"
)

// TestAsciiTransformerShortDst covers the transform.Transformer ErrShortDst
// contract branch. This path is unreachable from Ballerina source: x/text sizes
// its own destination buffer when encode/decode runs through url:encode/decode,
// so the short-destination case is only observable by calling Transform directly.
// All other url behaviour is exercised by the corpus tests in
// corpus/bal/library/subset2/url-*.bal.
func TestAsciiTransformerShortDst(t *testing.T) {
	t.Parallel()
	// Destination smaller than the ASCII source forces ErrShortDst once dst fills.
	dst := make([]byte, 2)
	nDst, nSrc, err := asciiTransformer{}.Transform(dst, []byte("abcdef"), false)
	if err != transform.ErrShortDst {
		t.Fatalf("Transform short dst: err = %v, want ErrShortDst", err)
	}
	if nDst != 2 || nSrc != 2 {
		t.Errorf("Transform short dst: nDst=%d nSrc=%d, want 2 and 2", nDst, nSrc)
	}
	if string(dst) != "ab" {
		t.Errorf("Transform short dst: dst = %q, want %q", dst, "ab")
	}
}
