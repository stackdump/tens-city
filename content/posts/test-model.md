---
title: 'Petri-net: Inhibitor and Weighted Arcs Example'
description: Petri-net with a single capacitated place, weighted arcs and inhibitor flags — used for tests and examples
datePublished: 2025-11-02T00:00:00Z
author:
  name: Tens City Team
  type: Organization
  url: https://tens.city
tags:
  - petri-net
  - model
  - test
collection: guides
lang: en
slug: petri-net-inhibitor-weighted
---


# Petri\-net: Inhibitor and Weighted Arcs Example

[![pflow](https://pflow.xyz/img/z4EBG9jE7roa3Th3FeUgXNfMYurHVW5G7YYqUkUJU6NrKcWrMAu.svg)](https://pflow.xyz/?cid=z4EBG9jE7roa3Th3FeUgXNfMYurHVW5G7YYqUkUJU6NrKcWrMAu.svg)

This document describes a compact Petri\-net encoded as JSON\-LD. It demonstrates a single place with capacity, weighted arcs, and arcs marked with the \`inhibitTransition\` flag. The object identifier is \`@id: z2xFpT8KDD7FU8tiWSMcB8n6dxJriy2PtZJrcyCwHkn9fmug732\` and version \`1.1\`.

## Raw structure summary

- Places: \`place0\`
- Transitions: \`txn0\`, \`txn1\`, \`txn2\`, \`txn3\`
- Token palette: \`https://pflow.xyz/tokens/black\`

## Place details

- \`place0\`:
  - Type: Place
  - Capacity: 3
  - Initial marking: 1 token
  - Coordinates: (x: 130, y: 207)
  - Offset: 0

## Transitions (positions)

- \`txn0\` — (x: 46, y: 116)
- \`txn1\` — (x: 227, y: 112)
- \`txn2\` — (x: 43, y: 307)
- \`txn3\` — (x: 235, y: 306)

## Arcs

List includes direction, weight and whether the arc is marked as inhibiting the target transition \`inhibitTransition\`.

1. \`txn0\` → \`place0\`
   - Weight: 1
   - \`inhibitTransition\`: false
   - (output arc from \`txn0\` that deposits 1 token to \`place0\`)

2. \`place0\` → \`txn1\`
   - Weight: 3
   - \`inhibitTransition\`: false
   - (input arc: \`txn1\` requires 3 tokens from \`place0\` to fire)

3. \`txn2\` → \`place0\`
   - Weight: 3
   - \`inhibitTransition\`: true
   - (arc from \`txn2\` to \`place0\` with the inhibitor flag set; interpretation depends on runtime semantics — recorded here as an arc with \`inhibitTransition=true\`)

4. \`place0\` → \`txn3\`
   - Weight: 1
   - \`inhibitTransition\`: true
   - (inhibitor arc: presence of tokens in \`place0\` may prevent \`txn3\` from firing)

## Initial marking

- M0: \`place0\` = 1

## Semantics notes

- Weighted arcs: some arcs carry weights greater than 1 (notably the arc \`place0\` → \`txn1\` with weight 3), so transitions requiring those inputs need the indicated number of tokens.
- Inhibitor arcs: arcs with \`inhibitTransition: true\` are recorded; typical semantics treat a place\->transition inhibitor as preventing the transition when the place has tokens. The file also contains an arc from transition\->place with \`inhibitTransition: true\` — this flag is preserved verbatim and should be interpreted by the execution semantics implemented by tests/tools.
- Capacity: \`place0\` has capacity 3, so the place is bounded by 3 tokens.

## Properties

- Bounded: Yes (max tokens limited by capacity = 3).
- Safe: No (capacity > 1, so multiple tokens are allowed).
- Determinism / liveness: Not fully determined here; behavior depends on transition enabling rules, weighted inputs, and the interpretation of inhibitor flags.

## Purpose and usage in tests

- Example for handling:
  - Weighted input/output arcs.
  - Inhibitor arc metadata.
  - Places with finite capacity and non\-safe markings.
- Tests should verify canonical serialization, correct interpretation of weights and inhibitor flags, and storage\/CID stability for this canonical example.