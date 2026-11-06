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

import ballerina/data.jsondata;
import ballerina/io;

type Book record {|
    @jsondata:Name {value: "book-title"}
    string title;
    @jsondata:Name {value: "published_year"}
    int year;
    string author;
|};

type Alias record {|
    string value;
|};

annotation Alias FieldName on record field;

type Movie record {|
    @FieldName {value: "movie-title"}
    string title;
|};

function show(anydata|error v) {
    if v is error {
        io:println("error: ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    string bookJson = string `{"book-title":"Dune","published_year":1965,"author":"Herbert"}`;
    Book|error b1 = jsondata:parseString(bookJson, {}, Book);
    show(b1);
    // @output {"title":"Dune","year":1965,"author":"Herbert"}
    Book|error b2 = jsondata:parseAsType({"book-title": "Emma", "published_year": 1815, author: "Austen"}, {}, Book);
    show(b2);
    // @output {"title":"Emma","year":1815,"author":"Austen"}
    Book|error b3 = jsondata:parseBytes(bookJson.toBytes(), {}, Book);
    show(b3);
    // @output {"title":"Dune","year":1965,"author":"Herbert"}
    Book|error b4 = jsondata:parseString(string `{"title":"Dune","published_year":1965,"author":"Herbert"}`, {}, Book);
    show(b4);
    // @output error: required field 'title' not present in JSON
    Book|error b5 = jsondata:parseString(string `{"book-title":"Dune","author":"Herbert"}`, {}, Book);
    show(b5);
    // @output error: required field 'year' not present in JSON
    Movie|error m1 = jsondata:parseString(string `{"title":"Arrival"}`, {}, Movie);
    show(m1);
    // @output {"title":"Arrival"}
}
