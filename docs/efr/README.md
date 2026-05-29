# Engineering Failure Reports

An **Engineering Failure Report (EFR)** documents an architectural or design
failure that accumulated undetected in a codebase: the context that allowed it
to form, the hypothesis that justified it at the time, the moment it became
visible, and — most importantly — the rule that would prevent the same class of
failure from recurring.

EFRs are not blame documents. They are pattern libraries. The value is the
*learning*, not the post-mortem.

## What qualifies as an EFR

An EFR is warranted when:

- A structural decision (model shape, layer boundary, source of truth) turned
  out to be incorrect, and the correction required significant rework.
- The failure was *invisible* during normal development — tests passed, the
  code compiled, CI was green — yet the system had a meaningful integrity
  problem.
- The root cause reveals a reusable heuristic that should change how the team
  approaches a class of decisions going forward.

An EFR is **not** warranted for:

- Runtime bugs caught by tests.
- Operational incidents (use a standard postmortem format for those).
- Minor refactors that revealed a poor naming choice.

## Format

```markdown
# EFR-NNNN: <Title>

**Status:** Open | Resolved  
**Date detected:** YYYY-MM-DD  
**Date resolved:** YYYY-MM-DD (if applicable)  
**Author:** <name>

---

## Context

What was the state of the system before the failure became visible? Describe
the environment, the timeline of additions, and any assumptions that were in
place.

## Hypothesis

What belief, explicit or implicit, justified the design at the time?

## Failure

What was the actual failure? Describe the structural problem concretely:
duplicate stacks, conflicting sources of truth, hidden coupling, etc.

## Why it went undetected

Why did normal quality signals (compilation, tests, CI) not catch this?
What made it invisible?

## Applied solution

What was done to resolve it? Be concrete about what was removed or consolidated.

## Learning

What rule, checkpoint, or habit would prevent this class of failure from
recurring? State it as an actionable heuristic, not a vague principle.
```

## Numbering

EFRs are numbered sequentially with four digits: `EFR-0001`, `EFR-0002`, …  
Filenames follow the same scheme: `EFR-0001.md`, `EFR-0002.md`, …

A resolved EFR is never edited retroactively. The *learning* section is the
canonical output; it should be absorbed into working practices, checklists, or
CLAUDE.md as appropriate.
