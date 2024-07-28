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
	"strconv"
	"strings"
	"sync"
)

func main() {

	// Flag variables.
	var dir *string
	var headers *string
	var output *string
	var outputFile *string
	var threshold *float64

	// Command-line flags.
	dir = flag.String("d", ".", "Directory containing CSV files")
	headers = flag.String("header", "username", "CSV headers to perform logarithmic function on (comma-separated)")
	output = flag.String("o", "console", "Output format: console, html, csv")
	outputFile = flag.String("f", "output.csv", "Output file path for CSV format")
	threshold = flag.Float64("t", -3.0, "Threshold for log proportion to identify anomalies")
	flag.Parse()

	// Check that the directory exists.
	if _, err := os.Stat(*dir); os.IsNotExist(err) {
		log.Printf("Directory %s does not exist", *dir)
		return
	}

	// Ensure that dir and headers are not empty.
	if *dir == "" || *headers == "" {
		log.Printf("A directory and headers are required")
		return
	}

	// Split headers into a slice and validate.
	headerList := strings.Split(*headers, ",")
	for _, header := range headerList {
		if strings.TrimSpace(header) == "" {
			log.Printf("Invalid header value: %s", header)
			return
		}
	}

	// Retrieve list of CSV files from the specified directory.
	files, err := getCSVFiles(*dir)
	if err != nil {
		log.Printf("Failed to get CSV files: %v", err)
		return
	}

	// Determine number of goroutines to use based on the number of files.
	numGoroutines := determineNumGoroutines(files)

	// Aggregate data across all files.
	aggregatedData, totalEntries, err := aggregateData(files, headerList, numGoroutines)
	if err != nil {
		log.Printf("Failed to aggregate data: %v", err)
		return
	}

	// Identify anomalies based on the specified threshold.
	results, err := identifyAnomalies(aggregatedData, totalEntries, *threshold)
	if err != nil {
		log.Printf("Failed to identify anomalies: %v", err)
		return
	}

	// Output results in the specified format.
	switch *output {
	case "console":
		printResultsConsole(results)
	case "html":
		printResultsHTML(results, *outputFile)
	case "csv":
		printResultsCSV(results, *outputFile)
	default:
		log.Printf("Unknown output format: %s", *output)
		return
	}
}

/** Retrieve list of CSV files from the specified directory. */
func getCSVFiles(dir string) ([]string, error) {
	var files []string

	// Walk through the directory to find all CSV files.
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

	// If the files slice is empty, return an error.
	if len(files) == 0 {
		return nil, fmt.Errorf("no CSV files found in directory %s", dir)
	}

	return files, err
}

/** Determine the number of goroutines to use based on the number and size of files. */
func determineNumGoroutines(files []string) int {
	// Use a simple heuristic to decide number of goroutines.
	numGoroutines := len(files) / 2
	if numGoroutines > 10 {
		numGoroutines = 10
	} else if numGoroutines < 1 {
		numGoroutines = 1
	}

	return numGoroutines
}

/** Aggregate data from all files for the specified headers. */
func aggregateData(files []string, headers []string, numGoroutines int) (map[string][][]string, int, error) {
	var wg sync.WaitGroup

	// Create channels to communicate between goroutines.
	fileChan := make(chan string, len(files))
	dataChan := make(chan map[string][][]string, len(files))

	// Create a map to store aggregated data.
	aggregatedData := make(map[string][][]string)

	// Create a mutex to synchronize access to totalEntries.
	var totalEntriesMutex sync.Mutex

	totalEntries := 0

	// Launch goroutines to process files concurrently.
	for i := 0; i < numGoroutines; i++ {
		// Launch goroutine to process files.
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Process files concurrently.
			for file := range fileChan {
				data, count, err := processFile(file, headers)
				if err != nil {
					log.Printf("Failed to process file %s: %v", file, err)
					continue
				}
				dataChan <- data
				totalEntriesMutex.Lock()
				totalEntries += count
				totalEntriesMutex.Unlock()
			}
		}()
	}

	// Send files to fileChan for processing.
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	wg.Wait()
	close(dataChan)

	// Aggregate data from all goroutines.
	for data := range dataChan {
		for key, rows := range data {
			aggregatedData[key] = append(aggregatedData[key], rows...)
		}
	}

	return aggregatedData, totalEntries, nil
}

/** Process a single CSV file and return the rows containing the specified headers. */
func processFile(file string, headers []string) (map[string][][]string, int, error) {
	f, err := os.Open(filepath.Clean(file))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	// Read the CSV file.
	reader := csv.NewReader(f)
	// Set the number of fields per record to -1 to read all fields.
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file %s: %w", file, err)
	}

	// Create a map to store aggregated data.
	data := make(map[string][][]string)
	count := 0

	// Aggregate data for each specified header.
	for _, header := range headers {
		// Find the index of the specified header.
		headerIndex := -1
		for i, h := range records[0] {
			if h == header {
				headerIndex = i
				break
			}
		}

		// If the header is not found, skip it.
		if headerIndex == -1 {
			log.Printf("header %s not found in file %s", header, file)
			continue
		}

		// Aggregate rows for the specified header.
		for _, record := range records[1:] {
			if len(record) <= headerIndex {
				log.Printf("Record length less than header index in file %s", file)
				continue
			}
			value := record[headerIndex]

			// Add the record to the map.
			data[value] = append(data[value], record)
			count++
		}
	}

	return data, count, nil
}

/*
* Identify anomalies based on the specified threshold.
 */
func identifyAnomalies(data map[string][][]string, totalEntries int, threshold float64) ([][]string, error) {
	var anomalies [][]string

	// Loop through the data and identify anomalies.
	for _, rows := range data {
		count := len(rows)
		proportion := float64(count) / float64(totalEntries)
		logProportion := math.Log10(proportion)

		// Check if the log proportion is below the threshold.
		if logProportion < threshold {
			for _, row := range rows {
				anomalies = append(anomalies, append([]string{fmt.Sprintf("%d", count), fmt.Sprintf("%f", logProportion)}, row...))
			}
		}
	}

	// Sort the anomalies slice by logProportion in ascending order.
	sort.Slice(anomalies, func(i, j int) bool {
		// Convert logProportion from string to float64 for comparison
		logProportionI, _ := strconv.ParseFloat(anomalies[i][1], 64)
		logProportionJ, _ := strconv.ParseFloat(anomalies[j][1], 64)
		return logProportionI < logProportionJ
	})

	return anomalies, nil
}

/** Print results in console format. */
func printResultsConsole(results [][]string) {
	for _, record := range results {
		fmt.Println(record)
	}
}

/** Print results in HTML format and write to a file. */
func printResultsHTML(results [][]string, outputFile string) {
	// Open the output file.
	f, err := os.Create(filepath.Clean(outputFile))
	if err != nil {
		log.Printf("Failed to create output file %s: %v", outputFile, err)
	}
	defer f.Close()

	// Parse the HTML template file.
	tmpl, err := template.ParseFiles("templates/mel.html")
	if err != nil {
		log.Printf("Failed to parse template file: %v", err)
	}

	// Execute the template with the results data.
	err = tmpl.Execute(f, results)
	if err != nil {
		log.Printf("Failed to execute template: %v", err)
	}

	fmt.Println("HTML file generated successfully.")
}

/** Print results in CSV format. */
func printResultsCSV(results [][]string, outputFile string) {
	f, err := os.Create(filepath.Clean(outputFile))
	if err != nil {
		log.Printf("Failed to create output file %s: %v", outputFile, err)
		return
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
