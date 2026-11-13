import ballerina/test;

@test:Config {runs: 2, minPassRate: 1.5}
function testHigh() {}

@test:Config {runs: 2, minPassRate: -0.5}
function testNeg() {}
