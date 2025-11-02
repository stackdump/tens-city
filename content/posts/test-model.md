---
title: Test Model
description: Default Petri\-net model used by tests — a small cyclic net for firing and reachability checks
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
slug: petri-net-default
---

[![pflow](https://pflow.xyz/img/z4EBG9jE7roa3Th3FeUgXNfMYurHVW5G7YYqUkUJU6NrKcWrMAu.svg)](https://pflow.xyz/?cid=z4EBG9jE7roa3Th3FeUgXNfMYurHVW5G7YYqUkUJU6NrKcWrMAu.svg)

# Default Petri\-net test model

This document describes the default, minimal Petri\-net used in tests. It is intentionally small and deterministic so unit tests can verify canonicalization, serialization, firing and simple reachability.

## Structure

- Places: P0, P1, P2  
- Transitions: T0, T1  
- Arcs:
  - P0 → T0
  - T0 → P1
  - P1 → T1
  - T1 → P2
  - P2 → T0  (creates a simple cycle)

## Initial marking

- M0: P0 = 1, P1 = 0, P2 = 0

## Firing rules (standard Petri\-net semantics)

- A transition is enabled when all its input places contain at least the required number of tokens (here all weights are 1).
- When a transition fires, it consumes tokens from its input places and produces tokens to its output places.
- With M0, T0 is enabled. Firing T0 yields marking M1: P0 = 0, P1 = 1, P2 = 0. Then T1 is enabled, firing T1 yields M2: P0 = 0, P1 = 0, P2 = 1. Firing T1 followed by T0 returns the token to P1 and P2 in sequence, maintaining a single-token cycle.

## Incidence matrix (token change per transition)

Rows = places [P0, P1, P2], columns = transitions [T0, T1]

- C = [
  [-1,  0],  # P0
  [ 1, -1],  # P1
  [ 0,  1],  # P2
]

## Properties

- Bounded: Yes (max tokens = 1).
- Safe: Yes (no place can hold more than 1 token in this model).
- Liveness: The cycle allows repeated firings; depending on initial marking, the reachable set is cyclic and deterministic.
- Purpose: Simple deterministic net used as the default test model for serialization, canonicalization, signature, and storage tests.

## Usage in tests

- Verify canonical serialization of the net structure and initial marking.
- Verify firing semantics produce expected reachable markings.
- Verify storage and CID generation remain stable for this canonical example.