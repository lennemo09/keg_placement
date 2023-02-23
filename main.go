package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type Grid struct {
	Rows   int
	Cols   int
	Tiles  [][]int
	Source [2]int
}

var bestAttempts = 0

func dfs(grid *Grid, row, col int, visited [][]bool, counted [][]bool) int {
	// mark tile as visited
	visited[row][col] = true

	// initialize number of shore tiles to 0
	numShores := 0

	// check adjacent tiles
	for _, dir := range [][]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		newRow := row + dir[0]
		newCol := col + dir[1]

		// check if adjacent tile is within bounds and not visited
		if newRow >= 0 && newRow < grid.Rows && newCol >= 0 && newCol < grid.Cols && !visited[newRow][newCol] {
			// if adjacent tile is path, continue DFS
			if grid.Tiles[newRow][newCol] == 0 {
				counted[newRow][newCol] = true
				numShores += dfs(grid, newRow, newCol, visited, counted)
			}
			// if adjacent tile is keg, increment number of accessible keg tiles
			if grid.Tiles[newRow][newCol] == 1 && !counted[newRow][newCol] {
				numShores++
				counted[newRow][newCol] = true
			}
		}
	}
	return numShores
}

func evaluate(grid *Grid) (int, [][]bool) {
	// perform DFS and return number of accessible keg tiles
	visited := make([][]bool, grid.Rows)
	for row := 0; row < grid.Rows; row++ {
		visited[row] = make([]bool, grid.Cols)
	}

	counted := make([][]bool, grid.Rows)
	for row := 0; row < grid.Rows; row++ {
		counted[row] = make([]bool, grid.Cols)
	}

	counted[grid.Source[0]][grid.Source[1]] = true
	return dfs(grid, grid.Source[0], grid.Source[1], visited, counted), counted
}

func simulatedAnnealing(grid *Grid, steps int) {
	// set initial temperature and cooling rate
	temp := 1.0
	cooling := 0.95

	// set initial number of shore tiles and best number of accessible keg tiles
	currentScore, _ := evaluate(grid)
	bestScore := currentScore

	// iterate for a fixed number of steps
	for i := 0; i < steps; i++ {
		var changedTiles [][]int
		// randomly modify the grid
		for i := 0; i < 2; i++ {
			row := rand.Intn(grid.Rows)
			col := rand.Intn(grid.Cols)
			changedTiles = append(changedTiles, []int{row, col, grid.Tiles[row][col]})
			if grid.Tiles[row][col] == 1 {
				grid.Tiles[row][col] = 0
			} else if (row != grid.Source[0] && col != grid.Source[1]) && grid.Tiles[row][col] == 0 {
				grid.Tiles[row][col] = 1
			}
		}

		// evaluate the new placement of keg tiles
		newScore, countedGrid := evaluate(grid)

		// calculate change in score
		delta := newScore - currentScore

		// accept or reject new placement based on change in score and temperature
		if delta > 0 || math.Exp(float64(delta)/temp) > rand.Float64() {
			currentScore = newScore
			if currentScore > bestScore {
				bestScore = currentScore
				visualizeGrid(grid, bestScore, countedGrid)
				bestAttempts += 1
			}
		} else {
			// undo changes to grid
			for _, tile := range changedTiles {
				row := tile[0]
				col := tile[1]
				val := tile[2]
				grid.Tiles[row][col] = val
			}
		}

		// update temperature
		temp *= cooling
	}

	fmt.Printf("Best number of shore tiles: %d\n", bestScore)
}

func upscaleImage(img *image.RGBA, upscaleFactor int) *image.RGBA {
	// Determine the target dimensions of the upscaled image.
	targetWidth := img.Bounds().Dx() * upscaleFactor
	targetHeight := img.Bounds().Dy() * upscaleFactor

	// Create a new RGBA image with the target dimensions.
	newImg := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Iterate over each pixel in the new image.
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			// Calculate the corresponding pixel location in the original image.
			srcX := x / upscaleFactor
			srcY := y / upscaleFactor

			// Get the color of the pixel at the calculated location in the original image.
			color := img.At(srcX, srcY)

			// Set the color of the corresponding pixel in the new image.
			newImg.Set(x, y, color)
		}
	}

	return newImg
}

func clearFolder(folderPath string) error {
	// get a list of all files and subdirectories in the folder
	files, err := filepath.Glob(filepath.Join(folderPath, "*"))
	if err != nil {
		return err
	}

	// delete all files and subdirectories
	for _, file := range files {
		err := os.RemoveAll(file)
		if err != nil {
			return err
		}
	}

	return nil
}

func visualizeGrid(grid *Grid, bestScore int, countedGrid [][]bool) {
	// Create an empty image with the same size as the grid
	img := image.NewRGBA(image.Rect(0, 0, grid.Cols, grid.Rows))
	// fmt.Println("##########################")
	// // Set water tiles to blue and land tiles to green
	// for i := 0; i < grid.Rows; i++ {
	// 	for j := 0; j < grid.Cols; j++ {
	// 		fmt.Print(grid.Tiles[i][j], " ")
	// 	}
	// 	fmt.Print("\n")
	// }

	// Set keg tiles to brown, path tiles to white, and blocked tiles to black or red
	for i := 0; i < grid.Rows; i++ {
		for j := 0; j < grid.Cols; j++ {
			if grid.Tiles[i][j] == 0 && countedGrid[i][j] {
				img.Set(j, i, color.RGBA{190, 190, 190, 255})
			} else if grid.Tiles[i][j] == 0 {
				img.Set(j, i, color.RGBA{20, 20, 20, 255})
			} else if grid.Tiles[i][j] == 1 && countedGrid[i][j] {
				img.Set(j, i, color.RGBA{185, 126, 31, 255})
			} else {
				img.Set(j, i, color.RGBA{177, 57, 14, 255})
			}
		}
	}

	// Save the image to a file
	file, err := os.Create(fmt.Sprintf("./grids/grid_%d_%d.png", bestAttempts, bestScore))
	if err != nil {
		panic(err)
	}
	defer file.Close()

	biggerImg := upscaleImage(img, 30)
	err = png.Encode(file, biggerImg)
	if err != nil {
		panic(err)
	}
}

func createRandomGrid(rows, cols int, source [2]int) *Grid {
	tiles := make([][]int, rows)
	for i := 0; i < rows; i++ {
		tiles[i] = make([]int, cols)
		for j := 0; j < cols; j++ {
			if rand.Float64() < 0.5 {
				tiles[i][j] = 0 // path tile
			} else {
				tiles[i][j] = 1 // keg tile
			}
		}
	}
	// Set the source tile
	tiles[source[0]][source[1]] = 0

	return &Grid{
		Rows:   rows,
		Cols:   cols,
		Tiles:  tiles,
		Source: source,
	}
}

// Compare returns true if the first string precedes the second one according to natural order
func naturalStringCompare(a, b string) bool {
	chunksA := chunkify(a)
	chunksB := chunkify(b)

	nChunksA := len(chunksA)
	nChunksB := len(chunksB)

	for i := range chunksA {
		if i >= nChunksB {
			return false
		}

		aInt, aErr := strconv.Atoi(chunksA[i])
		bInt, bErr := strconv.Atoi(chunksB[i])

		// If both chunks are numeric, compare them as integers
		if aErr == nil && bErr == nil {
			if aInt == bInt {
				if i == nChunksA-1 {
					// We reached the last chunk of A, thus B is greater than A
					return true
				} else if i == nChunksB-1 {
					// We reached the last chunk of B, thus A is greater than B
					return false
				}

				continue
			}

			return aInt < bInt
		}

		// So far both strings are equal, continue to next chunk
		if chunksA[i] == chunksB[i] {
			if i == nChunksA-1 {
				// We reached the last chunk of A, thus B is greater than A
				return true
			} else if i == nChunksB-1 {
				// We reached the last chunk of B, thus A is greater than B
				return false
			}

			continue
		}

		return chunksA[i] < chunksB[i]
	}

	return false
}

func readDirNaturalOrder(dirPath string) ([]fs.FileInfo, error) {
	// Read the directory.
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// Sort the file names in natural order.
	sort.Slice(files, func(i, j int) bool {
		return naturalStringCompare(files[i].Name(), files[j].Name())
	})

	return files, nil
}

func createGifFromFolder(folderPath, outputPath string, delay int) error {
	fmt.Println("Creating GIF...")
	// Read all the image files in the folder.
	files, err := readDirNaturalOrder(folderPath)
	if err != nil {
		return err
	}

	outGif := &gif.GIF{}

	// Iterate over each file in the folder.
	for _, file := range files {
		// Open the image file.
		filePath := filepath.Join(folderPath, file.Name())
		reader, err := os.Open(filePath)

		if err != nil {
			return err
		}

		imageData, _, err := image.Decode(reader)
		if err != nil {
			return err
		}

		buf := bytes.Buffer{}

		if err = gif.Encode(&buf, imageData, nil); err != nil {
			return err
		}

		inGif, err := gif.Decode(&buf)
		if err != nil {
			return err
		}
		reader.Close()

		outGif.Image = append(outGif.Image, inGif.(*image.Paletted))

		outGif.Delay = append(outGif.Delay, delay)
	}

	outGif.Delay[len(files)-1] *= 10

	output := outputPath

	f, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	gif.EncodeAll(f, outGif)

	fmt.Println("GIF created.")
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// set the size of the grid and the number of water tiles
	rows := 12
	cols := 12

	clearFolder("./grids/")

	// initialize a random grid with the specified number of water tiles
	grid := createRandomGrid(rows, cols, [2]int{11, 11})

	// run the simulated annealing algorithm on the grid for 10000 steps
	simulatedAnnealing(grid, 1000000)

	createGifFromFolder("./grids/", "./simulation.gif", 50)
}
