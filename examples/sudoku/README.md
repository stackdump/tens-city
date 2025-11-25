# Sudoku Petri Net Example

This example demonstrates how to model Sudoku puzzles using Petri nets with the `pflow-xyz/go-pflow` library.

## Overview

Sudoku is a constraint satisfaction puzzle where numbers must be placed in a grid such that:
- Each row contains unique values
- Each column contains unique values
- Each sub-grid (block) contains unique values

This example includes both **4x4** and **9x9** Sudoku puzzles, with support for **Colored Petri Nets** where token colors represent digits.

## Available Models

### Standard Petri Net Models

#### 4x4 Sudoku (`sudoku-4x4-simple.jsonld`)
- Simpler variant with 2x2 blocks
- Uses numbers 1-4
- Great for understanding the basic concepts

#### 9x9 Sudoku (`sudoku-9x9.jsonld`)
- Standard Sudoku with 3x3 blocks
- Uses numbers 1-9
- Classic puzzle format

### Colored Petri Net Model

#### 9x9 Colored Sudoku (`sudoku-9x9-colored.jsonld`)
- **Token colors represent digits 1-9**
- Each color has a unique hex value for visualization
- More elegant constraint modeling using color restrictions
- Places track available colors per row/column/block

## Running the Analyzer

```bash
# From repository root - run 9x9 standard (default)
go run examples/sudoku/cmd/main.go

# Run 4x4 puzzle
go run examples/sudoku/cmd/main.go -size 4x4

# Run 9x9 standard puzzle
go run examples/sudoku/cmd/main.go -size 9x9

# Run 9x9 Colored Petri Net model
go run examples/sudoku/cmd/main.go -size 9x9 -colored
```

## Petri Net Models

### Standard Petri Net Representation

- **Places**: Each cell in the Sudoku grid is represented as a place
  - Initial marking represents given numbers (clues)
  - Empty cells start with 0 tokens
  - Filled cells have 1 token representing the number

- **Transitions**: Represent valid number placements
  - Each transition encodes the logic for filling a cell with a specific number
  - Transitions fire only when Sudoku constraints are satisfied

- **Arcs**: Connect places to transitions
  - Define the flow of tokens (number assignments)
  - Encode constraint checking logic

### Colored Petri Net Representation

In the Colored Petri Net model:

- **Colors**: Define a color set `DIGIT` with 9 colors (d1-d9) representing digits 1-9
  - Each color has a unique hex value for visualization:
    - d1 (1): `#FF6B6B` (red)
    - d2 (2): `#4ECDC4` (teal)
    - d3 (3): `#45B7D1` (blue)
    - d4 (4): `#96CEB4` (green)
    - d5 (5): `#FFEAA7` (yellow)
    - d6 (6): `#DDA0DD` (plum)
    - d7 (7): `#98D8C8` (mint)
    - d8 (8): `#F7DC6F` (gold)
    - d9 (9): `#BB8FCE` (purple)

- **Places**: Each cell can hold one colored token
  - `initialMarking` specifies the color of pre-filled cells
  - Empty cells have no tokens

- **Constraints**: Enforced through color restrictions
  - Row constraint: Each row can have at most one token of each color
  - Column constraint: Each column can have at most one token of each color
  - Block constraint: Each 3x3 block can have at most one token of each color

### Example Puzzles

#### 4x4 Puzzle (`sudoku-4x4-simple.jsonld`)

**Initial State:**
```
1 _ _ _
_ _ 2 _
_ 3 _ _
_ _ _ 4
```

**Solution:**
```
1 2 4 3
3 4 2 1
2 3 1 4
4 1 3 2
```

#### 9x9 Puzzle (`sudoku-9x9.jsonld`)

**Initial State:**
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

**Solution:**
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

## Sudoku Constraints as Petri Nets

In a full implementation, constraints would be encoded as:

1. **Row Constraints**: Inhibitor arcs prevent duplicate numbers in a row
2. **Column Constraints**: Inhibitor arcs prevent duplicate numbers in a column
3. **Block Constraints**: Inhibitor arcs prevent duplicate numbers in a 2x2 block (for 4x4)

The Petri net model allows us to:
- **Visualize** the puzzle structure
- **Analyze** reachability (can we reach a solution?)
- **Verify** that solutions satisfy all constraints
- **Generate** valid solution paths through state space exploration

## Key Concepts

### Places (Cells)
- `cell_i_j`: Represents the cell at row i, column j
- `initial`: Initial token count (0 for empty, 1 for given number)
- `capacity`: Maximum tokens allowed (1 per cell)
- `label`: Human-readable description

### Transitions (Moves)
- `fill_i_j_with_n`: Transition to fill cell (i,j) with number n
- Only fires when constraints are satisfied
- Represents a valid move in the solving process

### Solution State
- `solved`: Place that receives a token when the puzzle is complete
- Represents the goal state of the Petri net

## Using go-pflow

This example provides a foundation for using the `go-pflow` library. The current implementation:
- Demonstrates the JSON-LD structure for Petri nets compatible with go-pflow
- Shows how to model Sudoku constraints as a Petri net
- Validates the solution manually to verify correctness

Future enhancements could leverage go-pflow for:
- Loading and parsing Petri net models programmatically
- Simulating token flow and transitions
- Analyzing reachability and state spaces
- Visualizing Petri net execution
- Automated solution finding through state space search

## References

- [pflow-xyz/go-pflow](https://github.com/pflow-xyz/go-pflow) - Petri net simulation library
- [Petri Nets for Sudoku](https://ceur-ws.org/Vol-3721/paper2.pdf) - Academic paper on modeling puzzles with Petri nets
- [pflow.xyz](https://pflow.xyz) - Interactive Petri net editor and visualizer

## Extending This Example

To create a more complete Sudoku solver:

1. **Add all constraint transitions**: Encode row, column, and block constraints
2. **Implement search**: Use Petri net reachability analysis to find solutions
3. **Add backtracking**: Model the search tree as transition sequences
4. **Scale to 9x9**: Extend the model to standard Sudoku puzzles

## Educational Value

This example demonstrates:
- **Formal modeling** of constraint satisfaction problems
- **Declarative representation** of game rules
- **State space exploration** for puzzle solving
- **Constraint propagation** through token flow
- **Integration** of Petri nets with modern Go applications

Sudoku Petri nets show how formal methods can be applied to everyday puzzles, making abstract concepts tangible and visual.
