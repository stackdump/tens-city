# Sudoku Petri Net Example

This example demonstrates how to model a Sudoku puzzle using Petri nets with the `pflow-xyz/go-pflow` library.

## Overview

Sudoku is a constraint satisfaction puzzle where numbers must be placed in a grid such that:
- Each row contains unique values
- Each column contains unique values
- Each sub-grid (block) contains unique values

This example uses a **4x4 Sudoku** (simpler than the traditional 9x9) to demonstrate the concepts clearly.

## Petri Net Model

### Representation

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

### Example Puzzle

The `sudoku-4x4-simple.jsonld` file contains a simple 4x4 Sudoku puzzle:

**Initial State:**
```
1 _ _ _
_ _ 2 _
_ 3 _ _
_ _ _ 4
```

**Solution:**
```
1 2 3 4
3 4 2 1
2 3 4 1
4 1 2 3
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
