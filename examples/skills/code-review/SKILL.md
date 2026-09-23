---
name: code-review
description: Review a code change for verifiable correctness, safety, and regression defects without editing it.
---

# Code review

Use when asked to review a patch, merge request, or proposed change. Do not edit
files or claim a test ran unless it actually ran.

1. Read the intended behavior, diff, callers, and tests. Limit inspection to
   affected interfaces before expanding into unrelated code.
2. Trace one concrete failing input for each suspected defect. Check error
   handling, preserved data, permission boundaries, and compatibility changes.
3. Prefer evidence from source or existing test output. If execution is not
   permitted, describe a reproduction without pretending it was executed.
4. Return findings by severity: `path:line`, triggering condition, user impact,
   and minimal correction. Separate confirmed defects from questions.
5. If there are no findings, report coverage and the main remaining uncertainty;
   do not invent a finding to fill a quota.
