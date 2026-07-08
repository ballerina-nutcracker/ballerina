---
name: run-jballerina
description: Run a given Ballerina source file with jBallerina to compare behaviour against this interpreter
---

Use this skill when you need to validate this interpreter's behaviour against the Java implementation of Ballerina (jBallerina).

- Run the given Ballerina source file with jBallerina using:

  ```bash
  bal run $file
  ```

- Replace `$file` with the path to the `.bal` source file under test.
- Compare stdout, stderr, exit status, compile errors, and runtime panics/errors with this interpreter's behaviour.
  - When it comes to errors validate you get the error in the same line numbers and that interpreter error is not an unimplemented or internal error.
  - For panics validate the line numbers in stack frame are the same.
  - For both don't worry about the error messages being different
- If `bal` is not available in `PATH`, report that jBallerina is not installed or not configured for the current environment.
