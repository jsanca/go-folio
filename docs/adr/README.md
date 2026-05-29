# Architecture Decision Records

An **Architecture Decision Record (ADR)** captures a significant architectural
choice: the context that forced the decision, what was decided, and the
trade-offs accepted. ADRs are written *at decision time* — not as retrospective
documentation — so the reasoning is preserved while it is still fresh.

## When to write an ADR

Write an ADR when:

- A technology, library, or protocol is chosen over one or more alternatives.
- A structural boundary (service split, layer contract, interface shape) is
  established.
- A non-obvious trade-off is accepted and the reasoning should survive the
  original author's memory.

Do **not** write an ADR for:

- Implementation details inside a single component.
- Decisions that can be reversed trivially (a config value, a minor dependency).
- Operational or tooling choices with no long-term structural consequence.

## Format

```markdown
# ADR-NNN: <Title>

**Status:** Proposed | Accepted | Deprecated | Superseded by ADR-NNN  
**Date:** YYYY-MM-DD  
**Author:** <name>

---

## Context

What situation forced a decision? Include constraints, existing state, and any
alternatives that were on the table.

## Decision

What was decided, stated plainly and completely.

## Rationale

Why this option over the alternatives? Address each significant trade-off.

## Consequences

What becomes easier? What becomes harder? What is left open?
```

## Numbering

ADRs are numbered sequentially: `ADR-001`, `ADR-002`, …  
Filenames follow the same scheme: `ADR-001.md`, `ADR-002.md`, …

An accepted ADR is never edited retroactively. If a decision is reversed, mark
it `Superseded by ADR-NNN` and write a new record explaining the change.
