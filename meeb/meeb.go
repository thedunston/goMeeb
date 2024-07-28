package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func main() {

	// Flag variables.
	var dir *string
	var header *string
	var output *string
	var outputFile *string
	var threshold *float64

	// Command-line flags.
	dir = flag.String("d", ".", "Directory containing CSV files")
	header = flag.String("header", "Path", "CSV header to perform logarithmic function on")
	output = flag.String("o", "console", "Output format: console, html, csv")
	outputFile = flag.String("f", "output.csv", "Output file path for CSV format")
	threshold = flag.Float64("t", -3.0, "Threshold for log proportion to identify anomalies")
	flag.Parse()

	// Ensure that dir is not empty.
	if *dir == "" {

		log.Printf("A directory is required")

		return

	}

	// Check that the directory exists.
	if _, err := os.Stat(*dir); os.IsNotExist(err) {

		log.Printf("Directory %s does not exist", *dir)

		return

	}

	// If the output file exists, prompt the user to overwrite it.
	if _, err := os.Stat(*outputFile); !os.IsNotExist(err) {

		log.Printf("Output file %s already exists. Overwrite?", *outputFile)

		var overwrite string

		// Prompt the user for confirmation.
		_, err = fmt.Scanln(&overwrite)
		if err != nil {

			log.Printf("Failed to read user input: %v", err)

			return

		}

		if strings.ToLower(overwrite) != "y" {

			return

		}

	}

	// Retrieve list of CSV files from the specified directory
	files, err := getCSVFiles(*dir)
	if err != nil {

		log.Printf("Failed to get CSV files: %v", err)

		return
	}

	// Determine number of goroutines to use based on the number of files
	numGoroutines := determineNumGoroutines(files)

	// Aggregate data across all files
	aggregatedData, totalEntries := aggregateData(files, *header, numGoroutines)

	// Identify anomalies based on the specified threshold
	results := identifyAnomalies(aggregatedData, totalEntries, *threshold)

	// Output results in the specified format
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

/** Retrieve list of CSV files from the specified directory*/
func getCSVFiles(dir string) ([]string, error) {

	var files []string

	// Get the CSV files in the directory.
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {

		if err != nil {

			return err

		}

		// Add the CSV file to the list.
		if !info.IsDir() && filepath.Ext(path) == ".csv" {

			files = append(files, path)

		}

		return nil

	})

	// Check that the list is not empty.
	if len(files) == 0 {

		return nil, fmt.Errorf("no CSV files found in directory %s", dir)

	}

	return files, err
}

/** Determine the number of goroutines to use based on the number of files.*/
func determineNumGoroutines(files []string) int {

	// Basic method to determine the number of goroutines.
	numGoroutines := len(files) / 2

	if numGoroutines > 10 {

		numGoroutines = 10

	} else if numGoroutines < 1 {

		numGoroutines = 1

	}

	return numGoroutines

}

/*
* Aggregate data from all files for the specified header.

This may not be needed since the processing is quite fast with testing on 200+ files.
*/
func aggregateData(files []string, header string, numGoroutines int) (map[string]int, int) {
	var wg sync.WaitGroup
	fileChan := make(chan string, len(files))
	dataChan := make(chan map[string]int, len(files))
	aggregatedData := make(map[string]int)
	var totalEntriesMutex sync.Mutex
	totalEntries := 0

	// Launch goroutines to process files concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				data, count := processFile(file, header)
				dataChan <- data
				totalEntriesMutex.Lock()
				totalEntries += count
				totalEntriesMutex.Unlock()
			}
		}()
	}

	// Send files to fileChan for processing
	for _, file := range files {

		fileChan <- file

	}
	close(fileChan)

	wg.Wait()

	close(dataChan)

	// Aggregate data from all goroutines
	for data := range dataChan {

		for key, count := range data {

			aggregatedData[key] += count

		}

	}

	return aggregatedData, totalEntries

}

/** Process a single CSV file and return the count of occurrences for the specified header */
func processFile(file, header string) (map[string]int, int) {

	// Open the CSV file
	f, err := os.Open(filepath.Clean(file))
	if err != nil {

		log.Printf("Failed to open file %s: %v", file, err)
		return nil, 0

	}
	defer f.Close()

	reader := csv.NewReader(f)

	// Allow any number of fields per record.
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {

		log.Printf("Failed to read file %s: %v", file, err)
		return nil, 0

	}

	// Find the index of the specified header
	headerIndex := -1
	for i, h := range records[0] {

		if h == header {

			headerIndex = i
			break

		}
	}

	// Check if there is a header.
	if headerIndex == -1 {

		log.Printf("Header %s not found in file %s", header, file)
		return nil, 0

	}

	// Create a map to store counts.
	data := make(map[string]int)

	count := 0

	// Aggregate counts for the specified header
	for _, record := range records[1:] {

		value := record[headerIndex]
		data[value]++
		count++

	}

	return data, count

}

/** Identify anomalies based on the specified threshold. */
func identifyAnomalies(data map[string]int, totalEntries int, threshold float64) [][]string {

	var anomalies [][]string

	// Loop through the data and identify anomalies.
	for value, count := range data {

		proportion := float64(count) / float64(totalEntries)
		logProportion := math.Log10(proportion)

		if logProportion < threshold {

			anomalies = append(anomalies, []string{fmt.Sprintf("%d", count), fmt.Sprintf("%f", logProportion), value})

		}

	}

	// Sort the anomalies slice by logProportion in ascending order
	sort.Slice(anomalies, func(i, j int) bool {

		return anomalies[i][1] < anomalies[j][1]

	})

	return anomalies

}

/** Print results to the console. */
func printResultsConsole(results [][]string) {

	for _, record := range results {

		fmt.Println(record)

	}

}

/** Print results in HTML format. */
func printResultsHTML(results [][]string, outputFile string) {

	// Open the output file.
	f, err := os.Create(filepath.Clean(outputFile))
	if err != nil {

		log.Fatalf("Failed to create output file %s: %v", outputFile, err)

	}
	defer f.Close()

	// Parse the HTML template file.
	tmpl, err := template.ParseFiles("templates/meeb.html")
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

		writer.Write(record)

	}

}
