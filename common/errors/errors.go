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

package errors

import "fmt"

type IndexOutOfBoundsError struct {
	index  int
	length int
}

func (e IndexOutOfBoundsError) Error() string {
	return fmt.Sprintf("Index %d out of bounds for length %d", e.index, e.length)
}

func (e IndexOutOfBoundsError) GetIndex() int {
	return e.index
}

func (e IndexOutOfBoundsError) GetLength() int {
	return e.length
}

func NewIndexOutOfBoundsError(index, length int) *IndexOutOfBoundsError {
	return &IndexOutOfBoundsError{
		index:  index,
		length: length,
	}
}
