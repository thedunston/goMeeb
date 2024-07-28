package main

/**

This program will read a csv file as a baseline to compare to other CSV files in the same format.

*/

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

type PathAvg struct {
	Path string
	Avg  float64
}

func main() {

	// Flag variables.
	var dir *string
	var baselineFile *string
	var header *string
	var output *string
	var outputFile *string

	// Command-line flags.
	dir = flag.String("d", ".", "Directory containing CSV files")
	baselineFile = flag.String("b", "", "Baseline CSV file (optional)")
	header = flag.String("header", "Path", "CSV header to filter on")
	output = flag.String("o", "console", "Output format: console, html, csv")
	outputFile = flag.String("f", "output.csv", "Output file path for CSV format")

	flag.Parse()

	// Check that the directory exists.
	if _, err := os.Stat(*dir); os.IsNotExist(err) {

		fmt.Printf("Directory %s does not exist", *dir)
		return

	}

	// Check that the baseline file is provided as an option.
	if *baselineFile == "" {

		fmt.Printf("Baseline file is required")
		return

	} else {

		// Check if the baseline file exists.
		if _, err := os.Stat(*baselineFile); os.IsNotExist(err) {

			fmt.Printf("Baseline file %s does not exist", *baselineFile)
			return

		}

	}

	// Check if the output file exits. If so, the prompt to overwrite it.
	if _, err := os.Stat(*outputFile); !os.IsNotExist(err) {

		fmt.Printf("Output file %s already exists. Overwrite? (y/n) ", *outputFile)

		var confirm string
		_, err = fmt.Scanln(&confirm)
		if err != nil {

			log.Printf("Failed to read user input: %v", err)

			return

		}

		if confirm != "y" {

			return

		}

	}

	// Create a map to store the header counts.
	var baselinetheHeaders map[string]struct{}

	// Check if baseline provided.
	var baselineProvided bool = false

	if *baselineFile != "" {

		baselineProvided = true

	}

	// Results will be stored in a slice of slices.
	results := compareFiles(*dir, baselinetheHeaders, *header, baselineProvided)

	switch *output {

	case "console":
		printResultsConsole(results)

	case "html":
		printResultsHTML(results, *outputFile)

	case "csv":
		printResultsCSV(results, *outputFile)

	default:
		log.Fatalf("Unknown output format: %s", *output)

	}

}

/** Reads a CSV file and returns a slice of rows. */
func readCSV(filePath string) ([][]string, error) {

	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {

		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {

		return nil, err

	}

	return records, nil

}

/** Extracts the header from the CSV files. */
func getHeaderFromCSV(records [][]string, header string) (map[string]struct{}, error) {

	// Check if the CSV file is empty.
	if len(records) < 1 {

		return nil, fmt.Errorf("CSV file is empty")

	}

	// Find the index of the specified header.
	headerIndex := -1

	// Loop through the headers and find the specified header.
	for i, h := range records[0] {

		// If the header is found, break out of the loop.
		if h == header {

			headerIndex = i
			break

		}

	}

	// If the header is not found, return an error.
	if headerIndex == -1 {

		return nil, fmt.Errorf("The header %s was not found in the CSV file", header)

	}

	// Create a map to store the thePaths.
	thePaths := make(map[string]struct{})

	// Loop through the records and add the thePaths to the map.
	for _, record := range records[1:] {

		if len(record) > headerIndex {

			thePaths[record[headerIndex]] = struct{}{}

		}

	}

	return thePaths, nil
}

/** Reads the baseline file and returns a set of the selected headers. */
func readBaseline(baselineFile string, header string) (map[string]struct{}, error) {

	records, err := readCSV(baselineFile)
	if err != nil {

		return nil, err

	}

	return getHeaderFromCSV(records, header)
}

/** Compares the selected headers in files in the directory against the baseline headersor calculates averages. */
func compareFiles(dir string, baselinetheHeaders map[string]struct{}, header string, baselineProvided bool) [][]string {
	// Read the files in the directory.
	files, err := os.ReadDir(dir)
	if err != nil {

		log.Fatal(err)

	}

	var results [][]string
	// Create a map to store the header counts.
	headerCounts := make(map[string]int)

	totalFiles := 0

	// Loop through the files in the directory.
	for _, file := range files {
		// Check if the file is a CSV file.
		if strings.HasSuffix(file.Name(), ".csv") {
			filePath := filepath.Join(dir, file.Name())
			records, err := readCSV(filePath)

			if err != nil {

				log.Printf("Error reading file %s: %v\n", filePath, err)
				continue

			}

			// Get the header from the CSV file.
			thePaths, err := getHeaderFromCSV(records, header)
			if err != nil {

				log.Printf("Error processing file %s: %v\n", filePath, err)
				continue

			}

			// Update the header counts.
			totalFiles++
			// Loop through the thePaths and update the headerCounts.
			for path := range thePaths {

				headerCounts[path]++

			}

		}

	}

	var headerAvgs []PathAvg
	// Calculate the average number of times each path appears.
	for h, count := range headerCounts {

		average := float64(count) / float64(totalFiles)
		headerAvgs = append(headerAvgs, PathAvg{Path: h, Avg: average})

	}

	// Sort the headerAvgs by average in descending order.
	sort.Slice(headerAvgs, func(i, j int) bool {

		return headerAvgs[i].Avg > headerAvgs[j].Avg

	})

	for _, ha := range headerAvgs {

		results = append(results, []string{fmt.Sprintf("%.2f", ha.Avg), ha.Path})

	}

	return results

}

/** Print results to the console. */
func printResultsConsole(results [][]string) {

	for _, record := range results {

		fmt.Println(record)

	}

}

/** Print results in HTML format */
func printResultsHTML(results [][]string, outputFile string) {

	// Open the output file.
	f, err := os.Create(filepath.Clean(outputFile))
	if err != nil {

		log.Fatalf("Failed to create output file %s: %v", outputFile, err)

	}
	defer f.Close()

	// Parse the HTML template file.
	tmpl, err := template.ParseFiles("templates/baseline.html")
	if err != nil {

		log.Fatalf("Failed to parse template file: %v", err)

	}

	// Execute the template with the results data.
	err = tmpl.Execute(f, results)
	if err != nil {

		log.Fatalf("Failed to execute template: %v", err)

	}

	fmt.Println("HTML file generated successfully.")

}

/** Print results in CSV format. */
func printResultsCSV(results [][]string, outputFile string) {

	f, err := os.Create(filepath.Clean(outputFile))

	if err != nil {

		log.Fatalf("Failed to create output file %s: %v", outputFile, err)

	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	for _, record := range results {

		err := writer.Write(record)
		if err != nil {

			log.Printf("Failed to write record to CSV: %v", err)

		}

	}

}
