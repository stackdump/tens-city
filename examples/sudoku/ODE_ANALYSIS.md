# Sudoku ODE Analysis: Tracking Solution Progress

This document demonstrates how the ODE-compatible Petri net model tracks solution progress through the `solved` place, following the pattern from go-pflow's tic-tac-toe example.

## Overview

The ODE (Ordinary Differential Equation) model uses **constraint collector transitions** that fire when a row, column, or block is completely filled with unique digits. Each collector sends a token to the `solved` place, making it possible to track progress toward a complete solution.

## Model Structure

### Key Components

1. **Cell Places (P##)**: 81 places representing the 9×9 grid
   - Token present = cell is empty
   - No token = cell is filled (clue or placed digit)

2. **History Places (_D#_##)**: 729 places (81 cells × 9 digits)
   - Token present = that digit is in that cell
   - Example: `_D5_03` has a token → digit 5 is at position (0,3)

3. **Digit Transitions (D#_##)**: 729 transitions for placing digits
   - Consumes cell token (marks cell as filled)
   - Creates history token (records which digit was placed)

4. **Constraint Collectors**: 27 transitions
   - 9 Row collectors: `Row0_Complete`, `Row1_Complete`, ..., `Row8_Complete`
   - 9 Column collectors: `Col0_Complete`, `Col1_Complete`, ..., `Col8_Complete`
   - 9 Block collectors: `Block00_Complete`, `Block01_Complete`, ..., `Block22_Complete`

5. **Solved Place**: Accumulator showing constraint satisfaction
   - Maximum capacity: 27 tokens
   - Token count = number of satisfied constraints
   - Fully solved puzzle = 27 tokens

## Progress Tracking Examples

### Example 1: Initial State (30 Clues Given)

**Initial Puzzle:**
```
5 3 _ | _ 7 _ | _ _ _
6 _ _ | 1 9 5 | _ _ _
_ 9 8 | _ _ _ | _ 6 _
------+-------+------
8 _ _ | _ 6 _ | _ _ 3
4 _ _ | 8 _ 3 | _ _ 1
7 _ _ | _ 2 _ | _ _ 6
------+-------+------
_ 6 _ | _ _ _ | 2 8 _
_ _ _ | 4 1 9 | _ _ 5
_ _ _ | _ 8 _ | _ 7 9
```

**Initial State Analysis:**
- **Tokens in `solved`**: 0
  - No constraints are fully satisfied yet
  - Some rows/columns have 3-5 digits, but none are complete

**Constraint Status:**
```
Rows:        [3/9] [4/9] [4/9] [4/9] [5/9] [4/9] [4/9] [5/9] [5/9]
Columns:     [4/9] [3/9] [2/9] [3/9] [6/9] [4/9] [2/9] [4/9] [5/9]
Blocks:      [5/9] [3/9] [1/9] [3/9] [4/9] [3/9] [2/9] [4/9] [4/9]
```

None of the 27 constraints are complete → `solved` has 0 tokens.

### Example 2: Partial Solution (After Filling 20 Cells)

**Partially Filled Puzzle:**
```
5 3 4 | 6 7 8 | 9 1 2  ← Row 0: COMPLETE
6 7 2 | 1 9 5 | 3 4 8  ← Row 1: COMPLETE
1 9 8 | 3 4 2 | 5 6 7  ← Row 2: COMPLETE
------+-------+------
8 5 9 | 7 6 1 | 4 2 3  ← Row 3: COMPLETE
4 2 6 | 8 5 3 | 7 9 1  ← Row 4: COMPLETE
7 1 3 | 9 2 4 | 8 5 6  ← Row 5: COMPLETE
------+-------+------
9 6 1 | 5 3 7 | 2 8 _
2 8 7 | 4 1 9 | 6 3 _
3 4 5 | 2 8 6 | 1 7 _
```

**Progress Analysis:**
- **Tokens in `solved`**: 18
  - 6 complete rows (Row0-Row5 complete)
  - 9 complete columns (all columns now have 8-9/9 digits)
  - 3 complete blocks (top 3 blocks: Block00, Block01, Block02)

**Constraint Collectors That Fired:**
```
✓ Row0_Complete    (tokens in _D1-9_0[0-8] all present)
✓ Row1_Complete    (tokens in _D1-9_1[0-8] all present)
✓ Row2_Complete    (tokens in _D1-9_2[0-8] all present)
✓ Row3_Complete
✓ Row4_Complete
✓ Row5_Complete
✓ Col0_Complete    (tokens in _D1-9_[0-8]0 all present)
✓ Col1_Complete
...all 9 columns...
✓ Block00_Complete (tokens in _D1-9_[0-2][0-2] all present)
✓ Block01_Complete
✓ Block02_Complete
```

**ODE Simulation Prediction:**
With 18/27 constraints satisfied and only 3 cells remaining, ODE simulation would show high probability of reaching 27/27 (fully solved).

### Example 3: Complete Solution

**Fully Solved Puzzle:**
```
5 3 4 | 6 7 8 | 9 1 2
6 7 2 | 1 9 5 | 3 4 8
1 9 8 | 3 4 2 | 5 6 7
------+-------+------
8 5 9 | 7 6 1 | 4 2 3
4 2 6 | 8 5 3 | 7 9 1
7 1 3 | 9 2 4 | 8 5 6
------+-------+------
9 6 1 | 5 3 7 | 2 8 4
2 8 7 | 4 1 9 | 6 3 5
3 4 5 | 2 8 6 | 1 7 9
```

**Final State Analysis:**
- **Tokens in `solved`**: 27 (maximum)
- **All constraint collectors fired:**
  - ✓ 9 Row collectors (all rows have digits 1-9)
  - ✓ 9 Column collectors (all columns have digits 1-9)
  - ✓ 9 Block collectors (all 3×3 blocks have digits 1-9)

**Verification:**
```bash
go run examples/sudoku/cmd/main.go -size 9x9 -ode
```

Output should show:
```
✓ All rows contain unique values
✓ All columns contain unique values
✓ All 3×3 blocks contain unique values

Petri Net Structure:
  Places: 811 (81 cells + 729 history + 1 solved)
  Transitions: 756 (729 digit placements + 27 constraint collectors)
  Arcs: ~7000+

ODE Analysis:
  Cell Places: 81
  History Places: 729
  Digit Transitions: 729
  Constraint Collectors: 27
  Solved Place: 'solved' (27/27 tokens)
```

## How Constraint Collectors Work

### Example: Row 0 Constraint Collector

**Transition:** `Row0_Complete`

**Input Arcs (81 total):**
All 9 history places for each position in row 0:
```
_D1_00 → Row0_Complete  (digit 1 at position 0,0)
_D2_00 → Row0_Complete  (digit 2 at position 0,0)
_D3_00 → Row0_Complete  (digit 3 at position 0,0)
...
_D1_01 → Row0_Complete  (digit 1 at position 0,1)
_D2_01 → Row0_Complete  (digit 2 at position 0,1)
...
_D1_08 → Row0_Complete  (digit 1 at position 0,8)
_D2_08 → Row0_Complete  (digit 2 at position 0,8)
_D9_08 → Row0_Complete  (digit 9 at position 0,8)
```

**Output Arc:**
```
Row0_Complete → solved  (weight: 1)
```

**Firing Condition:**
The transition fires when:
- All 81 input places have tokens (meaning all 9 cells in row 0 are filled)
- No duplicate digits exist (each digit 1-9 appears exactly once)

When `Row0_Complete` fires, it sends 1 token to the `solved` place.

### Example: Block 0,0 Constraint Collector

**Transition:** `Block00_Complete`

Checks the top-left 3×3 block:
```
[0,0] [0,1] [0,2]
[1,0] [1,1] [1,2]
[2,0] [2,1] [2,2]
```

**Input Arcs (81 total):**
```
_D1_00, _D2_00, ..., _D9_00 (9 possible digits at 0,0)
_D1_01, _D2_01, ..., _D9_01 (9 possible digits at 0,1)
...
_D1_22, _D2_22, ..., _D9_22 (9 possible digits at 2,2)
```

Fires when all 9 cells in this block contain unique digits 1-9.

## Using ODE Simulation with go-pflow

The ODE model is compatible with go-pflow for simulation. Here's how you would use it:

```go
package main

import (
    "github.com/pflow-xyz/go-pflow/pflow"
    "encoding/json"
    "fmt"
    "os"
)

func main() {
    // Load the ODE model
    data, _ := os.ReadFile("examples/sudoku/sudoku-9x9-ode.jsonld")
    
    var petriNet pflow.PetriNet
    json.Unmarshal(data, &petriNet)
    
    // Create ODE simulator
    sim := pflow.NewODESimulator(&petriNet)
    
    // Simulate forward in time
    results := sim.Simulate(100) // 100 time steps
    
    // Check solved place token count over time
    for t, state := range results {
        solvedTokens := state.Places["solved"]
        fmt.Printf("Time %d: Solved constraints: %d/27\n", t, solvedTokens)
        
        // Predict if solution is reachable
        if sim.PredictReachability("solved", 27) {
            fmt.Println("  → Solution is reachable!")
        }
    }
}
```

## Progress Metrics

The `solved` place provides several key metrics:

### 1. Constraint Satisfaction Rate
```
Progress = (tokens_in_solved / 27) × 100%

Examples:
  0 tokens  = 0%   (initial state)
  9 tokens  = 33%  (perhaps all rows satisfied)
  18 tokens = 67%  (rows + columns satisfied)
  27 tokens = 100% (fully solved)
```

### 2. Solution Proximity
The more constraints satisfied, the fewer valid moves remain:

| Solved Tokens | Remaining Constraints | Difficulty |
|---------------|----------------------|------------|
| 0-5           | 22-27                | High       |
| 6-15          | 12-21                | Medium     |
| 16-23         | 4-11                 | Low        |
| 24-26         | 1-3                  | Very Low   |
| 27            | 0                    | Complete   |

### 3. Deadlock Detection
If the ODE simulation shows that the `solved` place token count plateaus below 27, the puzzle may be:
- Incorrectly designed (no valid solution)
- In an invalid state (conflicting digit placements)
- Deadlocked (requires backtracking)

## Comparison with Tic-Tac-Toe

| Aspect            | Tic-Tac-Toe         | Sudoku 9×9           |
|-------------------|---------------------|----------------------|
| Grid Size         | 3×3 (9 cells)       | 9×9 (81 cells)       |
| Cell Places       | 9                   | 81                   |
| History Places    | 18 (9×2 players)    | 729 (81×9 digits)    |
| Pattern Collectors| 8 (win patterns)    | 27 (constraints)     |
| Accumulator Places| 2 (`win_x`,`win_o`) | 1 (`solved`)         |
| Max Tokens        | 1 per player        | 27 (all constraints) |
| Simulation Time   | ~1.8s per game      | ~minutes (larger)    |

## Practical Applications

### 1. Move Evaluation
Before placing a digit, simulate both options and compare `solved` token accumulation rates:
```
Place 5 at (4,3) → ODE shows +2 constraints satisfied
Place 7 at (4,3) → ODE shows +1 constraint satisfied
→ Choose 5 (better progress)
```

### 2. Difficulty Assessment
Analyze the initial puzzle:
```
Many constraints nearly complete → Easier puzzle
Few constraints close to complete → Harder puzzle
```

### 3. Solver Guidance
Use ODE gradient to guide a solver:
```
For each empty cell:
  For each valid digit:
    Simulate placement
    Measure Δ(solved tokens)
Pick move with highest Δ
```

## Running ODE Analysis

```bash
# Analyze 9×9 ODE model
go run examples/sudoku/cmd/main.go -size 9x9 -ode

# Compare with standard model
go run examples/sudoku/cmd/main.go -size 9x9

# Analyze 4×4 ODE model (simpler)
go run examples/sudoku/cmd/main.go -size 4x4 -ode
```

## Conclusion

The ODE-compatible model provides a powerful way to:
- **Track progress** via `solved` place tokens (0-27 scale)
- **Predict solutions** via ODE simulation
- **Evaluate moves** by measuring constraint satisfaction impact
- **Detect deadlocks** when token flow stops before 27

This approach mirrors go-pflow's tic-tac-toe win detection, scaled up to handle Sudoku's 27 constraints across 81 cells.
