import ballerina/test;

int beforeEachCount = 0;
int afterEachCount = 0;

@test:BeforeEach
function be() {
    beforeEachCount += 1;
}

@test:AfterEach
function ae() {
    afterEachCount += 1;
}

@test:Config {}
function testA() {
    test:assertFail("a fails");
}

@test:Config {
    dependsOn: [testA],
    runs: 2,
    minPassRate: 0.5
}
function testEvalSkippedByDependsOn() {
}

function setupFails() {
    test:assertFail("setup always fails");
}

@test:Config {
    before: setupFails,
    runs: 3,
    minPassRate: 0.5
}
function testEvalBeforeFails() {
}

// Named to sort alphabetically last among independent tests (no dependsOn
// on the above, which would cascade this test's own skip once testA
// fails) — testplan.BuildGraph orders independent tests by name, so this
// still runs after testA/testEvalSkippedByDependsOn/testEvalBeforeFails.
@test:Config {}
function testZCountsCheck() {
    // By the time this test's own body runs, its own BeforeEach has
    // already fired (3rd: testA, testEvalBeforeFails, this one), but not
    // yet its own AfterEach — so afterEachCount is still testA's alone. A
    // dependsOn-skipped evaluation contributes to neither count at all,
    // and a before:-failed evaluation contributes only one BeforeEach,
    // never an AfterEach — both confirmed against real jballerina.
    test:assertEquals(beforeEachCount, 3, msg = "beforeEachCount");
    test:assertEquals(afterEachCount, 1, msg = "afterEachCount");
}
