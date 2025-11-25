package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// SudokuPetriNet represents the structure of our Sudoku Petri net model
// This example demonstrates the JSON-LD structure for Petri nets compatible
// with go-pflow. Future enhancements could use go-pflow for simulation and
// state space analysis.
type SudokuPetriNet struct {
	Context     interface{}           `json:"@context"`
	Type        string                `json:"@type"`
	Version     string                `json:"@version"`
	Description string                `json:"description"`
	Puzzle      PuzzleInfo            `json:"puzzle"`
	Token       []string              `json:"token"`
	Colors      *ColorDefinition      `json:"colors,omitempty"`
	Places      map[string]Place      `json:"places"`
	Transitions map[string]Transition `json:"transitions"`
	Arcs        []Arc                 `json:"arcs"`
	Constraints *Constraints          `json:"constraints,omitempty"`
}

// ODEAnalysis tracks tokens flowing through constraint collectors
type ODEAnalysis struct {
	ConstraintCollectors int
	SolvedPlace          string
	HistoryPlaces        int
	CellPlaces           int
	DigitTransitions     int
}

type PuzzleInfo struct {
	Description      string  `json:"description"`
	Size             int     `json:"size"`
	BlockSize        int     `json:"block_size"`
	InitialState     [][]int `json:"initial_state"`
	Solution         [][]int `json:"solution"`
	ConstraintsCount int     `json:"constraints_count,omitempty"`
	ODECompatible    bool    `json:"ode_compatible,omitempty"`
}

// ColorDefinition represents the color set for Colored Petri Nets
type ColorDefinition struct {
	Description string       `json:"description"`
	ColorSet    string       `json:"colorSet"`
	Values      []ColorValue `json:"values"`
}

// ColorValue represents a single color in the color set
type ColorValue struct {
	ID    string `json:"id"`
	Value int    `json:"value"`
	Label string `json:"label"`
	Hex   string `json:"hex"`
}

// ColoredMarking represents a colored token marking
type ColoredMarking struct {
	Color string `json:"color"`
	Count int    `json:"count"`
}

type Place struct {
	Type           string           `json:"@type"`
	Label          string           `json:"label"`
	ColorSet       string           `json:"colorSet,omitempty"`
	Initial        []int            `json:"initial,omitempty"`
	InitialMarking []ColoredMarking `json:"initialMarking,omitempty"`
	Capacity       []int            `json:"capacity"`
	X              int              `json:"x"`
	Y              int              `json:"y"`
}

type Transition struct {
	Type  string `json:"@type"`
	Label string `json:"label"`
	Guard string `json:"guard,omitempty"`
	Role  string `json:"role,omitempty"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type Arc struct {
	Type        string `json:"@type"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	Weight      []int  `json:"weight"`
	Inscription string `json:"inscription,omitempty"`
}

// Constraints describes how Sudoku constraints are encoded
type Constraints struct {
	Description      string `json:"description"`
	RowConstraint    string `json:"row_constraint"`
	ColumnConstraint string `json:"column_constraint"`
	BlockConstraint  string `json:"block_constraint"`
	Implementation   string `json:"implementation"`
}

func main() {
	// Parse command line flags
	size := flag.String("size", "9x9", "Sudoku size: 4x4 or 9x9")
	colored := flag.Bool("colored", false, "Use Colored Petri Net model (colors represent digits)")
	ode := flag.Bool("ode", false, "Use ODE-compatible model (constraint collectors like tic-tac-toe)")
	flag.Parse()

	fmt.Println("Sudoku Petri Net Analyzer")
	fmt.Println("==========================")
	fmt.Println()

	// Determine which model file to use
	var modelFile string
	switch *size {
	case "4x4", "4":
		if *ode {
			modelFile = "sudoku-4x4-ode.jsonld"
		} else {
			modelFile = "sudoku-4x4-simple.jsonld"
		}
	case "9x9", "9":
		if *colored {
			modelFile = "sudoku-9x9-colored.jsonld"
		} else {
			modelFile = "sudoku-9x9.jsonld"
		}
	default:
		fmt.Printf("Error: Invalid size '%s'. Use 4x4 or 9x9\n", *size)
		os.Exit(1)
	}

	// Find the model file - try multiple possible locations
	possiblePaths := []string{
		modelFile, // Running from examples/sudoku
		filepath.Join("examples", "sudoku", modelFile),             // Running from repo root
		filepath.Join("..", "..", "examples", "sudoku", modelFile), // Running from examples/sudoku/cmd
	}

	modelPath := ""
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			modelPath = path
			break
		}
	}

	if modelPath == "" {
		fmt.Printf("Error: Could not find %s\n", modelFile)
		fmt.Println("Usage: Run from the repository root or examples/sudoku directory")
		os.Exit(1)
	}

	// Load the Petri net model
	data, err := os.ReadFile(modelPath)
	if err != nil {
		fmt.Printf("Error reading model file: %v\n", err)
		os.Exit(1)
	}

	var model SudokuPetriNet
	if err := json.Unmarshal(data, &model); err != nil {
		fmt.Printf("Error parsing model: %v\n", err)
		os.Exit(1)
	}

	// Determine puzzle size
	puzzleSize := model.Puzzle.Size
	if puzzleSize == 0 {
		puzzleSize = len(model.Puzzle.Solution)
	}
	blockSize := model.Puzzle.BlockSize
	if blockSize == 0 {
		blockSize = int(math.Sqrt(float64(puzzleSize)))
	}

	// Check model type
	isColored := model.Type == "ColoredPetriNet" || model.Colors != nil
	isODE := model.Puzzle.ODECompatible || *ode

	// Display puzzle information
	fmt.Println("Puzzle Information:")
	fmt.Printf("  Description: %s\n", model.Puzzle.Description)
	fmt.Printf("  Size: %dx%d\n", puzzleSize, puzzleSize)
	fmt.Printf("  Block Size: %dx%d\n", blockSize, blockSize)
	
	// Display model type
	if isODE {
		fmt.Println("  Model Type: ODE-Compatible Petri Net (like tic-tac-toe)")
	} else if isColored {
		fmt.Println("  Model Type: Colored Petri Net")
	} else {
		fmt.Println("  Model Type: Standard Petri Net")
	}
	fmt.Println()

	// Display color information for Colored Petri Nets
	if isColored && model.Colors != nil {
		fmt.Println("Color Set (DIGIT):")
		fmt.Println("  Colors represent Sudoku digits 1-9")
		for _, c := range model.Colors.Values {
			fmt.Printf("  • %s = %d (color: %s)\n", c.ID, c.Value, c.Hex)
		}
		fmt.Println()
	}

	// Display initial state
	fmt.Println("Initial State:")
	printGrid(model.Puzzle.InitialState, puzzleSize, blockSize)
	fmt.Println()

	// Display solution
	fmt.Println("Solution:")
	printGrid(model.Puzzle.Solution, puzzleSize, blockSize)
	fmt.Println()

	// Analyze the Petri net structure
	fmt.Println("Petri Net Structure:")
	fmt.Printf("  Type: %s\n", model.Type)
	fmt.Printf("  Places: %d\n", len(model.Places))
	fmt.Printf("  Transitions: %d\n", len(model.Transitions))
	fmt.Printf("  Arcs: %d\n", len(model.Arcs))
	fmt.Println()

	// ODE-specific analysis
	if isODE {
		odeAnalysis := analyzeODEModel(model)
		fmt.Println("ODE Analysis (tic-tac-toe style):")
		fmt.Printf("  Cell Places (P##): %d\n", odeAnalysis.CellPlaces)
		fmt.Printf("  History Places (_D#_##): %d\n", odeAnalysis.HistoryPlaces)
		fmt.Printf("  Digit Transitions (D#_##): %d\n", odeAnalysis.DigitTransitions)
		fmt.Printf("  Constraint Collectors: %d\n", odeAnalysis.ConstraintCollectors)
		fmt.Printf("  Solved Place: %s\n", odeAnalysis.SolvedPlace)
		fmt.Println()
		fmt.Println("ODE Win Detection Pattern:")
		fmt.Println("  1. Cell places (P##) hold tokens for empty cells")
		fmt.Println("  2. Digit transitions (D#_##) place digits and create history")
		fmt.Println("  3. History places (_D#_##) track which digit is in each cell")
		fmt.Println("  4. Constraint collectors fire when all cells in row/col/block filled")
		fmt.Println("  5. All constraints feed into 'solved' place")
		fmt.Println("  6. ODE simulation measures token flow to 'solved'")
		fmt.Println()
	}

	// Count given numbers (clues) - handle both colored and regular markings
	clues := 0
	emptyCells := 0
	for name, place := range model.Places {
		// Identify cell places using a helper function for clarity
		isCell := isCellPlace(name, place.Label)
		if !isCell {
			continue
		}
		
		if len(place.InitialMarking) > 0 {
			// Colored Petri Net marking
			clues++
		} else if len(place.Initial) > 0 && place.Initial[0] > 0 {
			// Standard Petri Net: initial=1 means clue present
			// ODE model: initial=1 means cell is EMPTY (token available for digit transition)
			// This inversion matches the tic-tac-toe pattern where cell places with tokens
			// are available for moves, and empty places are already filled
			if isODE {
				emptyCells++
			} else {
				clues++
			}
		} else {
			// Standard Petri Net: initial=0 means empty cell
			// ODE model: initial=0 means cell is already FILLED (no token = clue present)
			if isODE {
				clues++
			} else {
				emptyCells++
			}
		}
	}

	fmt.Println("Puzzle Statistics:")
	fmt.Printf("  Given clues: %d\n", clues)
	fmt.Printf("  Empty cells: %d\n", emptyCells)
	fmt.Printf("  Total cells: %d\n", clues+emptyCells)
	fmt.Println()

	// Verify solution
	fmt.Println("Solution Verification:")
	if verifySolution(model.Puzzle.Solution, puzzleSize, blockSize) {
		fmt.Println("  ✓ Solution is valid!")
		fmt.Println("  ✓ All rows contain unique values")
		fmt.Println("  ✓ All columns contain unique values")
		fmt.Printf("  ✓ All %dx%d blocks contain unique values\n", blockSize, blockSize)
	} else {
		fmt.Println("  ✗ Solution is invalid")
	}
	fmt.Println()

	// Display transition information
	if isODE {
		fmt.Println("Constraint Collectors (like tic-tac-toe win patterns):")
		for name, trans := range model.Transitions {
			if trans.Role == "constraint" {
				fmt.Printf("  - %s: %s\n", name, trans.Label)
			}
		}
		fmt.Println()
	} else {
		fmt.Println("Available Transitions:")
		for name, trans := range model.Transitions {
			fmt.Printf("  - %s: %s\n", name, trans.Label)
		}
		fmt.Println()
	}

	fmt.Println("This Petri net model demonstrates how Sudoku constraints")
	fmt.Println("can be encoded as places, transitions, and arcs.")
	fmt.Println()
	fmt.Println("Key Concepts:")
	if isODE {
		fmt.Println("  • Like tic-tac-toe: cells hold tokens, moves create history")
		fmt.Println("  • Constraint collectors fire when row/col/block is complete")
		fmt.Println("  • 'solved' place accumulates tokens from all collectors")
		fmt.Println("  • ODE simulation predicts solution feasibility")
		fmt.Println("  • Can use go-pflow for state space analysis")
	} else if isColored {
		fmt.Println("  • Places represent cells that can hold colored tokens")
		fmt.Println("  • Token colors represent Sudoku digits (1-9)")
		fmt.Println("  • Each cell can hold at most one colored token")
		fmt.Println("  • Row/Column/Block constraints ensure unique colors")
		fmt.Println("  • Transitions fire only when color constraints are satisfied")
	} else {
		fmt.Println("  • Places represent cell states")
		fmt.Println("  • Transitions represent valid moves")
		fmt.Println("  • Arcs enforce Sudoku constraints")
		fmt.Println("  • Token flow represents the solving process")
	}

	// Display constraint information for Colored Petri Nets
	if isColored && model.Constraints != nil {
		fmt.Println()
		fmt.Println("Colored Petri Net Constraints:")
		fmt.Printf("  Row: %s\n", model.Constraints.RowConstraint)
		fmt.Printf("  Column: %s\n", model.Constraints.ColumnConstraint)
		fmt.Printf("  Block: %s\n", model.Constraints.BlockConstraint)
	}
}

// isCellPlace identifies whether a place represents a Sudoku cell
// Cell places use these naming conventions:
// - Standard models: "Cell(row,col)" labels like "Cell(0,0)"
// - ODE models: "P##" names like "P00", "P01" (3 chars starting with 'P')
func isCellPlace(name, label string) bool {
	// Check label-based naming (standard and colored models)
	if strings.HasPrefix(label, "Cell(") {
		return true
	}
	// Check ODE naming convention: P followed by 2 digits (e.g., P00, P33)
	if len(name) == 3 && name[0] == 'P' && isDigit(name[1]) && isDigit(name[2]) {
		return true
	}
	return false
}

// isDigit checks if a byte is an ASCII digit
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// isDigitRole checks if a role string indicates a digit placement transition
// Digit roles are "d1", "d2", etc. (2 chars starting with 'd' followed by a digit)
func isDigitRole(role string) bool {
	if len(role) != 2 {
		return false
	}
	return role[0] == 'd' && isDigit(role[1])
}

// analyzeODEModel examines the Petri net structure for ODE-compatible patterns
// The ODE model follows the tic-tac-toe pattern from go-pflow:
// - Cell places (P##) hold tokens for available moves
// - History places (_D#_##) track which digit was placed
// - Constraint collectors fire when row/col/block is complete
// - Solved place accumulates tokens from all constraint collectors
func analyzeODEModel(model SudokuPetriNet) ODEAnalysis {
	analysis := ODEAnalysis{}
	
	for name, place := range model.Places {
		// Cell places are named P## (like P00, P01, etc.)
		if isCellPlace(name, place.Label) {
			analysis.CellPlaces++
		}
		// History places start with _D (like _D1_00)
		if strings.HasPrefix(name, "_D") {
			analysis.HistoryPlaces++
		}
		// Solved place
		if name == "solved" {
			analysis.SolvedPlace = place.Label
		}
	}
	
	for _, trans := range model.Transitions {
		// Constraint collectors have role "constraint"
		if trans.Role == "constraint" {
			analysis.ConstraintCollectors++
		}
		// Digit placement transitions have role like "d1", "d2", etc.
		if isDigitRole(trans.Role) {
			analysis.DigitTransitions++
		}
	}
	
	if analysis.SolvedPlace == "" {
		analysis.SolvedPlace = "solved"
	}
	
	return analysis
}

// printGrid displays a Sudoku grid with appropriate formatting
func printGrid(grid [][]int, size, blockSize int) {
	if len(grid) != size {
		fmt.Println("  Invalid grid size")
		return
	}

	// Build the separator line
	cellWidth := 2
	if size > 9 {
		cellWidth = 3
	}

	blockSep := "+"
	for b := 0; b < size/blockSize; b++ {
		for c := 0; c < blockSize; c++ {
			blockSep += fmt.Sprintf("%s+", repeatStr("-", cellWidth+1))
		}
	}

	fmt.Printf("  %s\n", blockSep)
	for i, row := range grid {
		fmt.Print("  |")
		for j, val := range row {
			if val == 0 {
				fmt.Printf(" %s|", repeatStr("_", cellWidth-1))
			} else {
				fmt.Printf(" %*d|", cellWidth-1, val)
			}
			// Add block separator
			if (j+1)%blockSize == 0 && j+1 < size {
				fmt.Print("|")
			}
		}
		fmt.Println()
		// Add block separator row
		if (i+1)%blockSize == 0 {
			fmt.Printf("  %s\n", blockSep)
		} else if i+1 < size {
			// Regular row separator
			rowSep := "+"
			for b := 0; b < size/blockSize; b++ {
				for c := 0; c < blockSize; c++ {
					rowSep += fmt.Sprintf("%s+", repeatStr("-", cellWidth+1))
				}
			}
			fmt.Printf("  %s\n", rowSep)
		}
	}
}

func repeatStr(s string, count int) string {
	return strings.Repeat(s, count)
}

// verifySolution checks if a Sudoku solution is valid
func verifySolution(grid [][]int, size, blockSize int) bool {
	if len(grid) != size {
		return false
	}

	// Check rows
	for i := 0; i < size; i++ {
		if !isUnique(grid[i], size) {
			return false
		}
	}

	// Check columns
	for j := 0; j < size; j++ {
		col := make([]int, size)
		for i := 0; i < size; i++ {
			col[i] = grid[i][j]
		}
		if !isUnique(col, size) {
			return false
		}
	}

	// Check blocks
	for blockRow := 0; blockRow < size/blockSize; blockRow++ {
		for blockCol := 0; blockCol < size/blockSize; blockCol++ {
			block := make([]int, 0, blockSize*blockSize)
			for i := 0; i < blockSize; i++ {
				for j := 0; j < blockSize; j++ {
					block = append(block, grid[blockRow*blockSize+i][blockCol*blockSize+j])
				}
			}
			if !isUnique(block, size) {
				return false
			}
		}
	}

	return true
}

// isUnique checks if all values in a slice are unique and in range 1-size
func isUnique(values []int, size int) bool {
	seen := make(map[int]bool)
	for _, v := range values {
		if v < 1 || v > size {
			return false
		}
		if seen[v] {
			return false
		}
		seen[v] = true
	}
	return len(seen) == size
}
