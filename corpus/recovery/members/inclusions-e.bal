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

type ObjectBase object {
    int inheritedValue;
};
type RecordBase record {
    int inheritedValue;
};
type NotStructured int;

class MixedClass {
    *MissingClassBase; // @error
    *NotStructured; // @error
    *ObjectBase;
    function read() returns int => inheritedValue + missingClassReadValue; // @error
}
type MixedObject object {
    *MissingObjectBase; // @error
    *NotStructured; // @error
    *ObjectBase;
    int localValue;
};
class ObjectConsumer {
    *MixedObject;
    function read() returns int => inheritedValue + localValue;
}
type MixedRecord record {
    *MissingRecordBase; // @error
    *NotStructured; // @error
    *RecordBase;
    int localValue;
};
function readClass(MixedClass value) returns int => value.read();
function readObject(MixedObject value) returns int => value.inheritedValue;
function readRecord(MixedRecord value) returns int => value.inheritedValue;
function readConsumer(ObjectConsumer value) returns int => value.read();
