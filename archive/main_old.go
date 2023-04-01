package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <csvFile>\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	fileData, err := getFileData()
	if err != nil {
		exitGracefully(err)
	}

	if _, err := isValidFile(fileData.filepath); err != nil {
		exitGracefully(err)
	}

	writerChannel := make(chan map[string]interface{})
	done := make(chan bool)

	go processCSVFile(*fileData, writerChannel)
	go writeJSONFile(fileData.filepath, writerChannel, done, fileData.pretty)

	<-done
}

type inputFile struct {
	filepath  string
	separator string
	pretty    bool
}

func getFileData() (*inputFile, error) {
	if len(os.Args) < 2 {
		return nil, errors.New("A filepath argument is required")
	}

	separator := flag.String("separator", "comma", "Column separator")
	pretty := flag.Bool("pretty", false, "Generate pretty JSON")

	flag.Parse()

	fileLocation := flag.Arg(0)

	if !(*separator == "comma" || *separator == "semicolon") {
		return nil, errors.New("only comma or semicolon separators are allowed")
	}

	return &inputFile{fileLocation, *separator, *pretty}, nil
}

func isValidFile(filename string) (bool, error) {
	if fileExt := filepath.Ext(filename); fileExt != ".csv" {
		return false, fmt.Errorf("file %s is not CSV", filename)
	}

	if _, err := os.Stat(filename); err != nil && os.IsNotExist(err) {
		return false, fmt.Errorf("file %s does not exist", filename)
	}

	return true, nil
}

func processCSVFile(fileData inputFile, writerChan chan<- map[string]interface{}) {
	file, err := os.Open(fileData.filepath)
	check(err)
	defer file.Close()

	var headers, line []string

	reader := csv.NewReader(file)
	reader.LazyQuotes = true

	if fileData.separator == "semicolon" {
		reader.Comma = ';'
	}

	headers, err = reader.Read()
	check(err)

	for {
		line, err = reader.Read()
		if err == io.EOF {
			close(writerChan)
			break
		} else if err != nil {
			exitGracefully(err)
		}

		record, err := processLine(headers, line)
		if err != nil {
			fmt.Printf("Line: %sError: %s\n", line, err)
			continue
		}

		writerChan <- record
	}
}

func check(e error) {
	if e != nil {
		exitGracefully(e)
	}
}

func exitGracefully(err error) {
	_, err = fmt.Fprintf(os.Stderr, "error: %v\n", err)
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(1)
}

func processLine(headers []string, dataList []string) (map[string]interface{}, error) {
	if len(dataList) != len(headers) {
		return nil, errors.New("line doesn't match headers format. Skipping")
	}

	recordMap := make(map[string]interface{})
	for i, name := range headers {
		recordMap[strings.TrimSpace(name)] = strings.Trim(dataList[i], "\n\" ")
	}

	return recordMap, nil
}

func writeJSONFile(csvPath string, writerChannel <-chan map[string]interface{}, done chan<- bool, pretty bool) {
	writeString := createStringWriter(csvPath, pretty) // Instantiating a JSON writer function
	jsonFunc, breakLine := getJSONFunc(pretty)         // Instantiating the JSON parse function and the breakline character
	// Log for informing
	fmt.Println("Writing JSON file...")
	// Writing the first character of our JSON file. We always start with a "[" since we always generate array of record
	writeString("["+breakLine, false)
	first := true
	for {
		// Waiting for pushed records into our writerChannel
		record, more := <-writerChannel
		if more {
			if !first { // If it's not the first record, we break the line
				writeString(","+breakLine, false)
			} else {
				first = false // If it's the first one, we don't break the line
			}

			jsonData := jsonFunc(record) // Parsing the record into JSON
			writeString(jsonData, false) // Writing the JSON string with our writer function
		} else { // If we get here, it means there aren't more record to parse. So we need to close the file
			writeString(breakLine+"]", true) // Writing the final character and closing the file
			fmt.Println("Completed!")        // Logging that we're done
			done <- true                     // Sending the signal to the main function so it can correctly exit out.
			break                            // Stopping the for-loop
		}
	}
}

func createStringWriter(csvPath string, pretty bool) func(string, bool) {
	jsonDir := filepath.Dir(csvPath)
	var jsonName string
	if pretty {
		jsonName = fmt.Sprintf("%s.json", strings.TrimSuffix(filepath.Base(csvPath), ".csv")+"-pretty")
	} else {
		jsonName = fmt.Sprintf("%s.json", strings.TrimSuffix(filepath.Base(csvPath), ".csv")+"-compact")
	}
	finalLocation := filepath.Join(jsonDir, jsonName)

	f, err := os.Create(finalLocation)
	check(err)

	return func(data string, close bool) {
		_, err := f.WriteString(data)
		check(err)

		if close {
			f.Close()
		}
	}
}

func getJSONFunc(pretty bool) (func(map[string]interface{}) string, string) {
	// Declaring the variables we're going to return at the end
	var jsonFunc func(map[string]interface{}) string
	var breakLine string
	if pretty { //Pretty is enabled, so we should return a well-formatted JSON file (multi-line)
		breakLine = "\n"
		jsonFunc = func(record map[string]interface{}) string {
			jsonData, _ := json.MarshalIndent(record, "   ", "   ") // By doing this we're ensuring the JSON generated is indented and multi-line
			return "   " + string(jsonData)                         // Transforming from binary data to string and adding the indent characets to the front
		}
	} else { // Now pretty is disabled so we should return a compact JSON file (one single line)
		breakLine = "" // It's an empty string because we never break lines when adding a new JSON object
		jsonFunc = func(record map[string]interface{}) string {
			jsonData, _ := json.Marshal(record) // Now we're using the standard Marshal function, which generates JSON without formating
			return string(jsonData)             // Transforming from binary data to string
		}
	}

	return jsonFunc, breakLine // Returning everythinbg
}
