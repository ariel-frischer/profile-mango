---
name: test-driven-coding
description: Implement a scoped behavior change with a failing test, minimal code, and focused validation.
---

# Test-driven coding

Use for a bug fix or small feature when the repository provides a test command.
Follow repository-specific instructions and do not assume any language or runner.

1. State the observable input, expected result, and smallest affected interface.
2. Add or select one focused regression test. Run it and confirm the failure is
   caused by the requested behavior, not a broken environment.
3. Implement the minimum change; rerun the test and inspect failures before
   changing more code. Cover the relevant negative or boundary case.
4. Run surrounding checks that match the edited surface. Do not run destructive
   or live integration commands merely because a script exists.
5. Report changed behavior, the exact checks and results, and any unverified
   interaction. If a failing-first test was impractical, state why.
