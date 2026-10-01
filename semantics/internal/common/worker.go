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

package common

import "github.com/ballerina-nutcracker/ballerina/semtypes"

// WorkerFailureType is the error part of a worker's declared return type,
// given the worker's future<T> type: what a message action gets when the peer
// fails instead of doing its part.
func WorkerFailureType(cx semtypes.Context, workerTy semtypes.SemType) semtypes.SemType {
	return semtypes.Intersect(semtypes.FutureEventualType(cx, workerTy), semtypes.Error)
}
