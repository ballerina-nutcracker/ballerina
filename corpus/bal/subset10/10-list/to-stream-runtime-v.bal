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

import ballerina/io;

function printNext(stream<int, ()> values) {
    record {| int value; |}|() next = values.next();
    if next is record {| int value; |} {
        io:println(next.value);
    } else {
        io:println("done");
    }
}

public function main() {
    int[] replaced = [10, 20];
    stream<int, ()> replacedStream = replaced.toStream();
    printNext(replacedStream); // @output 10
    replaced[0] = 11;
    replaced[1] = 21;
    replaced.push(30);
    printNext(replacedStream); // @output 21
    printNext(replacedStream); // @output done

    int[] beforeAppend = [1];
    stream<int, ()> beforeAppendStream = beforeAppend.toStream();
    beforeAppend.push(2);
    printNext(beforeAppendStream); // @output 1
    printNext(beforeAppendStream); // @output 2
    printNext(beforeAppendStream); // @output done

    int[] afterAppend = [3];
    stream<int, ()> afterAppendStream = afterAppend.toStream();
    printNext(afterAppendStream); // @output 3
    afterAppend.push(4);
    printNext(afterAppendStream); // @output done

    int[] beforeGrowth = [5];
    stream<int, ()> beforeGrowthStream = beforeGrowth.toStream();
    beforeGrowth[3] = 8;
    printNext(beforeGrowthStream); // @output 5
    printNext(beforeGrowthStream); // @output 0
    printNext(beforeGrowthStream); // @output 0
    printNext(beforeGrowthStream); // @output 8
    printNext(beforeGrowthStream); // @output done

    int[] afterGrowth = [9];
    stream<int, ()> afterGrowthStream = afterGrowth.toStream();
    printNext(afterGrowthStream); // @output 9
    afterGrowth[3] = 12;
    printNext(afterGrowthStream); // @output done

    int[] empty = [];
    stream<int, ()> emptyStream = empty.toStream();
    printNext(emptyStream); // @output done
    empty.push(13);
    printNext(emptyStream); // @output done

    int[] exhausted = [14];
    stream<int, ()> exhaustedStream = exhausted.toStream();
    printNext(exhaustedStream); // @output 14
    printNext(exhaustedStream); // @output done
    exhausted.push(15);
    printNext(exhaustedStream); // @output done

    int[] closed = [16, 17];
    stream<int, ()> closedStream = closed.toStream();
    printNext(closedStream); // @output 16
    () firstClose = closedStream.close();
    () secondClose = closedStream.close();
    io:println(firstClose is (), secondClose is ()); // @output truetrue
    printNext(closedStream); // @output done
    closed.push(18);
    printNext(closedStream); // @output done
}
