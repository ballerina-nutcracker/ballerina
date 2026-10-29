# Reporting language and dependency-library limitations upstream

Some limitations discovered while porting or extending a stdlib aren't local to the package being
worked on — they're one of:

- **A language/interpreter limitation** — a construct or behaviour the compiler/BIR/interpreter
  pipeline doesn't support yet, beyond what's already catalogued in `AGENTS.md`. Surfaces via
  `add-stdlib-support` Step 3's "Handling unexpected compile failures", or an equivalent panic or
  compile error hit while implementing a gap.
- **A genuine bug in an already-ported dependency stdlib** — not a documented `Not Yet Supported` /
  `Partially Supported` / `Cannot Support` row (those are known gaps, already tracked by that
  package's README support matrix), but an actual defect: the dependency's Go implementation
  diverges from its own documented behaviour, or produces wrong output for a case its README claims
  is `Supported`.
- **A bug in the jBallerina reference implementation** — jBallerina's behaviour contradicts its spec or API docs, or is clearly defective, and the port implements the correct behaviour instead (see `add-stdlib-support` Step 5). These are tracked in the upstream library repo, not this one; see "jBallerina bugs" below.

Both are worth tracking independently of whatever local workaround this port ends up taking, so the
underlying limitation doesn't get silently rediscovered by the next person who hits it.

## What NOT to report here

- A gap already listed in a README's support matrix (`Not Yet Supported`, `Partially Supported`,
  `Cannot Support`) — already tracked there; no new issue needed.
- An architectural Go/JVM divergence recorded in **Notable Behavioural Changes** — that's an
  intentional, permanent design difference, not a bug.
- Ambiguity in the *jBallerina* reference itself (conflicting comments, unclear spec) — raise that
  with the developer per the skill's normal ask-don't-assume rule; it isn't a defect in this repo.

## Process

1. **Draft, don't file.** Never run `gh issue create` or otherwise open the issue yourself — write
   the summary and hand it to the developer to review. They may have more context (a duplicate may
   already exist, or the "bug" may be expected for a reason not visible from this port).
2. Tell the developer it belongs at **https://github.com/ballerina-nutcracker/ballerina/issues**.
3. Draft the issue using this template:

   ```markdown
   ## Title
   <One line: "<component>: <short symptom>", e.g. "interpreter: readonly & intersection on tuples not supported">

   ## Component
   - [ ] Language / interpreter (compiler, BIR, runtime)
   - [ ] Dependent stdlib: `ballerina/<name>`

   ## Summary
   <1-3 sentences: what's broken or missing, and how it was discovered — e.g. "found while porting
   ballerina/<new-name>, which needs <construct/behaviour>".>

   ## Reproduction
   \`\`\`ballerina
   <minimal .bal snippet that triggers it>
   \`\`\`
   Run with: `go run ./cli/cmd run <file>.bal`

   ## Expected vs actual
   - Expected: <what jBallerina does, or the documented/intended behaviour>
   - Actual: <panic message / compile error / wrong output, verbatim>

   ## Workaround applied in this port
   <e.g. "scoped out as Not Yet Supported", "moved to Go native", "none — this port is blocked on it">
   ```

4. If the developer confirms the draft, they file it themselves — or they may explicitly ask you to
   run `gh issue create` against `ballerina-nutcracker/ballerina` on their behalf. Filing an issue is
   visible to others, so only do that with direct confirmation for that specific issue; never file
   proactively.

## jBallerina bugs

A `jBallerina bug` row from the Step 5 behaviour table is tracked at **https://github.com/ballerina-platform/ballerina-library/issues** with the labels `Type/Bug` and `module/<name>`. The same draft-don't-file rule applies: hand the draft to the developer, and run `gh issue create` only with their direct confirmation for that specific issue. Search the repo for an existing issue first.

Use the repo's bug-report sections:

```markdown
### Description
<What jBallerina does, why it is wrong (quote the spec or API doc it contradicts), and the Java/.bal location, e.g. "`Cache.java` lines 58–63".>

### Steps to Reproduce
\`\`\`ballerina
<the run-jballerina probe>
\`\`\`
Output:
\`\`\`
<verbatim jBallerina output>
\`\`\`
Expected: <the correct output>.

Suggested fix: <one line, if obvious>.

### Version
ballerina/<name> <jBallerina package version>, Ballerina <bal version>

### Environment Details (with versions)
<OS, Ballerina version>
```

Once filed, link the issue from the package README's **Notable Behavioural Changes** bullet and from the corpus test that asserts the corrected behaviour.
