# Test framework design

## Requirements

- We will initially support a simplified subset of `ballerina/test` to validate the design
  - No special packages -> no special visibility rules
  - Only `enable`, `before` and `after` configurations
  - Support filtering (jBallerina: `--tests fn1,fn2` with `module:fn` and `*` wildcards)
  - No support for mocking
  - Support only `assertFail` in testerina
    ```ballerina
    function assertFail(string msg) returns never
    ```

## Design
- Test function will be annotated with `Config` annotation that can be attached to functions
```ballerina
// ballerina/test lib
type TestConfig record {|
  boolean enable = true
  function() returns ((any|error)) before?
  function() returns ((any|error)) after?
|};

public annotation TestConfig Config on function;
```
- We will introduce a new command `test` (similar to `build` and `run`). When you run `bal test` this will use a additional compiler plugin (in addition to what we normally include for `run` and `build`) when a module import `ballerina/test`
  - Ideally I would like this keep the definition of this in a separate package but I don't think this can be in the declaration of `ballerina/test` (we don't want project api to depend on libararies) perhaps a separate `testarina` module?
  - For normal `run`, `build` we treat the annotation as no-op 

- This compiler plugin reads all these annoatiations and build a `TestPlan` struct which is roughly
```go
type TestPlan struct {
  ModuleTests map[ModuleKey][]TestFunction // ModuleKey is org/module, the runtime lookup identity
}

type FunctionRef struct {
  Org, Module, Name string // empty means nothing; before/after may live in another module
}

type TestFunction struct {
  Name string
  Enabled bool
  Before FunctionRef
  After FunctionRef
}
```

- After we have build these struct (ideally I would like to keep this in the CLI side so I assume allocate the struct wrap it in the compiler plugin and pass it in to project api) we continue until we have loaded all the BIR packages to runtime (I think we can engage the compiler plugin just before desuger). CLI should get the runtime at this point so it can manually start executing functions

- It should then do `init` fallowed by start `listening` in a non-blocking manner
  - So any services that tests may depend on are up

- Then it builds an execution graph for tests: an ordered list of steps (modules in topological order, tests in source order, filtered by `--tests`), each step being before -> test -> after. Flat list for now; becomes a DAG when `dependsOn` is added

- Then we start executing each test showing the status in a nice form (use something like bubletee if needed)
  - I think current runtime API's give you what you need to implement this but we need to seperate the test data, execution graph and execution (last on top of current API)
  - Show progress updating and at the end only show test failures

- I imagine PAL needs to give a way to indicate testFailure that `assertFail` implementation can use
  - And CLI given it creates the PAL and inject a call back via PAL to detect test failures
