// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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
import ballerina/jwt;

function show([jwt:Header, jwt:Payload]|jwt:Error result) {
    if result is jwt:Error {
        io:println(result.message(), " (cause: ", result.cause() is error, ")");
    } else {
        io:println(result[0], " ", result[1]);
    }
}

public function main() {
    [jwt:Header, jwt:Payload]|jwt:Error decoded = jwt:decode("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImsxIiwieDV0IjoidCJ9.eyJpc3MiOiJ3c28yIiwic3ViIjoiYWxpY2UiLCJhdWQiOlsiYSIsImIiXSwiZXhwIjoxNzAwMDAwMDAwLCJuYmYiOiIxNjAwMDAwMDAwIiwiaWF0IjoxLjUsImp0aSI6ImoiLCJudW0iOjEuNSwiZSI6MWUyLCJvYmoiOnsieiI6MSwiYSI6W3RydWUsbnVsbCwicyJdfX0.sig");
    show(decoded); // @output {"alg":"HS256","typ":"JWT","kid":"k1","x5t":"t"} {"iss":"wso2","sub":"alice","aud":["a","b"],"exp":1700000000,"nbf":1600000000,"iat":0,"jti":"j","num":1.5,"e":1E+2,"obj":{"z":1,"a":[true,null,"s"]}}
    if decoded is [jwt:Header, jwt:Payload] {
        io:println(decoded[1]["num"] is decimal, " ", decoded[1]["e"] is decimal, " ", decoded[1]?.aud is string[]); // @output true true true
    }
    show(jwt:decode("eyJhbGciOiJSUzM4NCIsImN0eSI6IkpXVCJ9.eyJhdWQiOiJzaW5nbGUifQ")); // @output {"alg":"RS384","cty":"JWT"} {"aud":"single"}
    show(jwt:decode("eyJhbGciOiJSUzUxMiJ9.eyJzdWIiOiJ4In0.")); // @output {"alg":"RS512"} {"sub":"x"}
    show(jwt:decode("eyJhbGciOiJIUzM4NCJ9.eyJzdWIiOiJ4In0=")); // @output {"alg":"HS384"} {"sub":"x"}
    show(jwt:decode("eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOlsxLCJiIl19")); // @output {"alg":"HS256"} {"aud":["1","b"]}

    show(jwt:decode("a.b.c.d")); // @output Invalid JWT. (cause: false)
    show(jwt:decode("abc")); // @output Invalid JWT. (cause: false)
    show(jwt:decode("")); // @output Invalid JWT. (cause: false)
    show(jwt:decode("eyJhbGciOiJIUzUxMiJ9..sig")); // @output String to JSON conversion failed for JWT paylaod. (cause: true)
    show(jwt:decode("a$b.c")); // @output Base64 URL decode failed for JWT header. (cause: true)
    show(jwt:decode("eyJhbGciOiJIUzI1NiJ9.a$b")); // @output Base64 URL decode failed for JWT payload. (cause: true)
    show(jwt:decode("bm90IGpzb24.e30")); // @output String to JSON conversion failed for JWT header. (cause: true)
    show(jwt:decode("eyJhbGciOiJIUzI1NiJ9IHt9.e30")); // @output String to JSON conversion failed for JWT header. (cause: true)
    show(jwt:decode("eyJhbGciOiJIUzI1NiJ9.eyJuIjoxMjM0NTY3ODkwMTIzNDU2Nzg5MH0")); // @output String to JSON conversion failed for JWT paylaod. (cause: true)
    show(jwt:decode("__4.e30")); // @output Failed to convert byte[] of decoded header to string. (cause: true)
    show(jwt:decode("eyJhbGciOiJub25lIn0.eyJzdWIiOiJ4In0")); // @output Unsupported signing algorithm 'none'. (cause: false)
    show(jwt:decode("eyJhbGciOiJFUzI1NiJ9.e30")); // @output Unsupported signing algorithm 'ES256'. (cause: false)

    // jBallerina panics with TypeCastError for these: https://github.com/ballerina-platform/ballerina-library/issues/9709
    show(jwt:decode("WzEsMl0.e30")); // @output JWT header is not a JSON object. (cause: false)
    show(jwt:decode("eyJhbGciOiJIUzI1NiJ9.InMi")); // @output JWT payload is not a JSON object. (cause: false)
    show(jwt:decode("eyJhbGciOiJIUzI1NiIsInR5cCI6MX0.e30")); // @output Invalid value for the JWT header field 'typ'. A string is expected. (cause: false)
    show(jwt:decode("eyJhbGciOiJIUzI1NiIsImtpZCI6ZmFsc2V9.e30")); // @output Invalid value for the JWT header field 'kid'. A string is expected. (cause: false)
    show(jwt:decode("eyJhbGciOiJIUzI1NiJ9.eyJpc3MiOjV9")); // @output Invalid value for the JWT payload field 'iss'. A string is expected. (cause: false)
    show(jwt:decode("eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOnt9fQ")); // @output Invalid value for the JWT payload field 'aud'. A string is expected. (cause: false)
}
