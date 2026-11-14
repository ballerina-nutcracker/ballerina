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

import ballerina/data.yaml;
import ballerina/io;

type Server record {|
    string host;
    int port;
    boolean secure;
    string[] tags;
    decimal? weight;
|};

type Config record {|
    string name;
    Server[] servers;
|};

public function main() returns error? {
    string document = string `name: demo
servers:
  - &base
    host: localhost
    port: 8080
    secure: false
    tags: [api, "v1"]
    weight: null
  - host: example.com
    port: 443
    secure: true
    tags:
      - web
    weight: 1.5`;

    // Parse into a typed record, and into json.
    Config|yaml:Error config = yaml:parseString(document);
    io:println(config);
    json untyped = check yaml:parseString("a: [1, 2.5, true, null, text]");
    io:println(untyped);

    // Anchors and aliases.
    map<int> aliased = check yaml:parseString("x: &v 3\ny: *v");
    io:println(aliased);

    // A YAML stream of several documents binds to an array.
    int[] documents = check yaml:parseString("--- 1\n--- 2\n--- 3", {}, IntArray);
    io:println(documents);

    // Bytes and byte-block streams.
    map<string> fromBytes = check yaml:parseBytes("key: value".toBytes());
    io:println(fromBytes);
    byte[][] blocks = ["first: ".toBytes(), "1".toBytes()];
    map<int> fromStream = check yaml:parseStream(blocks.toStream());
    io:println(fromStream);

    // Write YAML back.
    io:println(check yaml:toYamlString({name: "demo", ports: [80, 443], meta: {secure: true}}));
    io:println(check yaml:toYamlString({name: "demo", ports: [80, 443]}, {flowStyle: true}));

    // Errors are yaml:Error values.
    int[]|yaml:Error bad = yaml:parseString("[1, 2");
    io:println(bad);
}

type IntArray int[];

// @output {"name":"demo","servers":[{"host":"localhost","port":8080,"secure":false,"tags":["api","v1"],"weight":null},{"host":"example.com","port":443,"secure":true,"tags":["web"],"weight":1.5}]}
// @output {"a":[1,2.5,true,null,"text"]}
// @output {"x":3,"y":3}
// @output [1,2,3]
// @output {"key":"value"}
// @output {"first":1}
// @output name: demo
// @output ports:
// @output - 80
// @output - 443
// @output meta:
// @output   secure: true
// @output {name: demo, ports: [80, 443]}
// @output error Error ("'unexpected event' at line: '1' column: '4'")
