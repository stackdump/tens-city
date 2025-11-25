# Sudoku Petri Net Example

This example demonstrates how to model Sudoku puzzles using Petri nets with the `pflow-xyz/go-pflow` library.

## Overview

Sudoku is a constraint satisfaction puzzle where numbers must be placed in a grid such that:
- Each row contains unique values
- Each column contains unique values
- Each sub-grid (block) contains unique values

This example includes both **4x4** and **9x9** Sudoku puzzles, with support for:
- **Colored Petri Nets** where token colors represent digits
- **ODE-Compatible Models** structured like the go-pflow tic-tac-toe example for solution detection

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

### ODE-Compatible Model (like tic-tac-toe)

#### 4x4 ODE Sudoku (`sudoku-4x4-ode.jsonld`)
- **Structured like the go-pflow tic-tac-toe example**
- Uses pattern collector transitions for constraint detection
- Win detection through token accumulation in `solved` place
- Compatible with go-pflow ODE simulation for solution prediction

**Model Structure:**
```
Cell Places (P##)  ──>  Digit Transitions (D#_##)  ──>  History Places (_D#_##)
                                                              │
                                                              v
                            Constraint Collectors ──> solved place
                     (Row/Column/Block Complete)
```

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

# Run 4x4 ODE-compatible model (tic-tac-toe style)
go run examples/sudoku/cmd/main.go -size 4x4 -ode
```

## ODE-Compatible Model (tic-tac-toe style)

The ODE model follows the same pattern as the tic-tac-toe example in go-pflow:

### Pattern Overview

1. **Cell Places (P00-P33)**: Represent empty cells, hold tokens when cell is available
2. **Digit Transitions (D#_##)**: Fire to place a digit, consume cell token and create history
3. **History Places (_D#_##)**: Record which digit is in each cell (like _X00, _O00 in tic-tac-toe)
4. **Constraint Collectors**: Transitions that fire when all cells in a row/column/block are filled
   - `Row0_Complete`, `Row1_Complete`, etc. (4 row collectors)
   - `Col0_Complete`, `Col1_Complete`, etc. (4 column collectors)
   - `Block00_Complete`, `Block01_Complete`, etc. (4 block collectors)
5. **Solved Place**: Accumulates tokens from all 12 constraint collectors

### ODE Win Detection

Just like tic-tac-toe uses ODE simulation to predict win likelihood by measuring token flow to `win_x` and `win_o`, Sudoku can use ODE simulation to:

- **Measure solution progress**: Token count in `solved` place indicates how many constraints are satisfied
- **Predict solution feasibility**: ODE simulation shows if current state leads to full solution
- **Evaluate moves**: Compare different digit placements by their effect on `solved` token accumulation

### Usage with go-pflow

```go
import (
    "github.com/pflow-xyz/go-pflow/parser"
    "github.com/pflow-xyz/go-pflow/engine"
)

// Load the model
jsonData, _ := os.ReadFile("examples/sudoku/sudoku-4x4-ode.jsonld")
net, _ := parser.FromJSON(jsonData)

// Run ODE simulation
eng := engine.New(net)
eng.RunODE(3.0)  // Simulate for t=3.0

// Check 'solved' place token count
state := eng.GetState()
solvedTokens := state["solved"]
fmt.Printf("Constraints satisfied: %.0f/12\n", solvedTokens)
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
- **ODE model follows tic-tac-toe pattern for solution detection**

Future enhancements could leverage go-pflow for:
- Loading and parsing Petri net models programmatically
- Simulating token flow and transitions
- Analyzing reachability and state spaces
- Visualizing Petri net execution
- Automated solution finding through state space search
- **ODE simulation for move evaluation (like tic-tac-toe AI)**

## References

- [pflow-xyz/go-pflow](https://github.com/pflow-xyz/go-pflow) - Petri net simulation library
- [go-pflow tic-tac-toe example](https://github.com/pflow-xyz/go-pflow/tree/main/examples/tictactoe) - ODE-based AI pattern
- [Petri Nets for Sudoku](https://ceur-ws.org/Vol-3721/paper2.pdf) - Academic paper on modeling puzzles with Petri nets
- [pflow.xyz](https://pflow.xyz) - Interactive Petri net editor and visualizer

## Extending This Example

To create a more complete Sudoku solver:

1. **Add all constraint transitions**: Encode row, column, and block constraints
2. **Implement search**: Use Petri net reachability analysis to find solutions
3. **Add backtracking**: Model the search tree as transition sequences
4. **Scale to 9x9**: Extend the ODE model to standard Sudoku puzzles
5. **ODE-based AI**: Use go-pflow ODE simulation to evaluate moves like tic-tac-toe

## Educational Value

This example demonstrates:
- **Formal modeling** of constraint satisfaction problems
- **Declarative representation** of game rules
- **State space exploration** for puzzle solving
- **Constraint propagation** through token flow
- **ODE analysis** for solution prediction (like tic-tac-toe)
- **Integration** of Petri nets with modern Go applications

Sudoku Petri nets show how formal methods can be applied to everyday puzzles, making abstract concepts tangible and visual.
