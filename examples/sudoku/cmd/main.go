package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Places      map[string]Place      `json:"places"`
	Transitions map[string]Transition `json:"transitions"`
	Arcs        []Arc                 `json:"arcs"`
}

type PuzzleInfo struct {
	Description  string  `json:"description"`
	InitialState [][]int `json:"initial_state"`
	Solution     [][]int `json:"solution"`
}

type Place struct {
	Type     string `json:"@type"`
	Label    string `json:"label"`
	Initial  []int  `json:"initial"`
	Capacity []int  `json:"capacity"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
}

type Transition struct {
	Type  string `json:"@type"`
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
}

type Arc struct {
	Type   string `json:"@type"`
	Source string `json:"source"`
	Target string `json:"target"`
	Weight []int  `json:"weight"`
}

func main() {
	fmt.Println("Sudoku Petri Net Analyzer")
	fmt.Println("==========================")
	fmt.Println()

	// Find the model file - try multiple possible locations
	possiblePaths := []string{
		"sudoku-4x4-simple.jsonld",                                    // Running from examples/sudoku
		filepath.Join("examples", "sudoku", "sudoku-4x4-simple.jsonld"), // Running from repo root
		filepath.Join("..", "..", "examples", "sudoku", "sudoku-4x4-simple.jsonld"), // Running from examples/sudoku/cmd
	}
	
	modelPath := ""
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			modelPath = path
			break
		}
	}
	
	if modelPath == "" {
		fmt.Println("Error: Could not find sudoku-4x4-simple.jsonld")
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

	// Display puzzle information
	fmt.Println("Puzzle Information:")
	fmt.Printf("  Description: %s\n", model.Puzzle.Description)
	fmt.Println()

	// Display initial state
	fmt.Println("Initial State:")
	printGrid(model.Puzzle.InitialState)
	fmt.Println()

	// Display solution
	fmt.Println("Solution:")
	printGrid(model.Puzzle.Solution)
	fmt.Println()

	// Analyze the Petri net structure
	fmt.Println("Petri Net Structure:")
	fmt.Printf("  Places: %d\n", len(model.Places))
	fmt.Printf("  Transitions: %d\n", len(model.Transitions))
	fmt.Printf("  Arcs: %d\n", len(model.Arcs))
	fmt.Println()

	// Count given numbers (clues)
	clues := 0
	emptyCells := 0
	for _, place := range model.Places {
		if len(place.Initial) > 0 && place.Initial[0] > 0 {
			clues++
		} else if len(place.Initial) > 0 && place.Initial[0] == 0 {
			emptyCells++
		}
	}

	fmt.Println("Puzzle Statistics:")
	fmt.Printf("  Given clues: %d\n", clues)
	fmt.Printf("  Empty cells: %d\n", emptyCells)
	fmt.Printf("  Total cells: %d\n", clues+emptyCells)
	fmt.Println()

	// Verify solution
	fmt.Println("Solution Verification:")
	if verifySolution(model.Puzzle.Solution) {
		fmt.Println("  ✓ Solution is valid!")
		fmt.Println("  ✓ All rows contain unique values")
		fmt.Println("  ✓ All columns contain unique values")
		fmt.Println("  ✓ All 2x2 blocks contain unique values")
	} else {
		fmt.Println("  ✗ Solution is invalid")
	}
	fmt.Println()

	// Display transition information
	fmt.Println("Available Transitions:")
	for name, trans := range model.Transitions {
		fmt.Printf("  - %s: %s\n", name, trans.Label)
	}
	fmt.Println()

	fmt.Println("This Petri net model demonstrates how Sudoku constraints")
	fmt.Println("can be encoded as places, transitions, and arcs.")
	fmt.Println()
	fmt.Println("Key Concepts:")
	fmt.Println("  • Places represent cell states")
	fmt.Println("  • Transitions represent valid moves")
	fmt.Println("  • Arcs enforce Sudoku constraints")
	fmt.Println("  • Token flow represents the solving process")
}

// printGrid displays a 4x4 Sudoku grid
func printGrid(grid [][]int) {
	if len(grid) != 4 || len(grid[0]) != 4 {
		fmt.Println("  Invalid grid size")
		return
	}

	fmt.Println("  +---+---+---+---+")
	for i, row := range grid {
		fmt.Print("  |")
		for _, val := range row {
			if val == 0 {
				fmt.Print(" _ |")
			} else {
				fmt.Printf(" %d |", val)
			}
		}
		fmt.Println()
		if i < 3 {
			fmt.Println("  +---+---+---+---+")
		}
	}
	fmt.Println("  +---+---+---+---+")
}

// verifySolution checks if a 4x4 Sudoku solution is valid
func verifySolution(grid [][]int) bool {
	if len(grid) != 4 {
		return false
	}

	// Check rows
	for i := 0; i < 4; i++ {
		if !isUnique(grid[i]) {
			return false
		}
	}

	// Check columns
	for j := 0; j < 4; j++ {
		col := make([]int, 4)
		for i := 0; i < 4; i++ {
			col[i] = grid[i][j]
		}
		if !isUnique(col) {
			return false
		}
	}

	// Check 2x2 blocks
	for blockRow := 0; blockRow < 2; blockRow++ {
		for blockCol := 0; blockCol < 2; blockCol++ {
			block := make([]int, 0, 4)
			for i := 0; i < 2; i++ {
				for j := 0; j < 2; j++ {
					block = append(block, grid[blockRow*2+i][blockCol*2+j])
				}
			}
			if !isUnique(block) {
				return false
			}
		}
	}

	return true
}

// isUnique checks if all values in a slice are unique and in range 1-4
func isUnique(values []int) bool {
	seen := make(map[int]bool)
	for _, v := range values {
		if v < 1 || v > 4 {
			return false
		}
		if seen[v] {
			return false
		}
		seen[v] = true
	}
	return len(seen) == 4
}
