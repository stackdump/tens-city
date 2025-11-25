package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
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
	Size         int     `json:"size"`
	BlockSize    int     `json:"block_size"`
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
	// Parse command line flags
	size := flag.String("size", "9x9", "Sudoku size: 4x4 or 9x9")
	flag.Parse()

	fmt.Println("Sudoku Petri Net Analyzer")
	fmt.Println("==========================")
	fmt.Println()

	// Determine which model file to use
	var modelFile string
	switch *size {
	case "4x4", "4":
		modelFile = "sudoku-4x4-simple.jsonld"
	case "9x9", "9":
		modelFile = "sudoku-9x9.jsonld"
	default:
		fmt.Printf("Error: Invalid size '%s'. Use 4x4 or 9x9\n", *size)
		os.Exit(1)
	}

	// Find the model file - try multiple possible locations
	possiblePaths := []string{
		modelFile,                                                  // Running from examples/sudoku
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

	// Display puzzle information
	fmt.Println("Puzzle Information:")
	fmt.Printf("  Description: %s\n", model.Puzzle.Description)
	fmt.Printf("  Size: %dx%d\n", puzzleSize, puzzleSize)
	fmt.Printf("  Block Size: %dx%d\n", blockSize, blockSize)
	fmt.Println()

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
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
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
