package csv

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type InputFile struct {
	Filepath  string
	separator string
	Pretty    bool
}

func GetFileData(args []string, pretty bool, separator string) (*InputFile, error) {

	fileLocation := args[0]

	if !(separator == "comma" || separator == "semicolon") {
		return nil, errors.New("only comma or semicolon separators are allowed")
	}

	return &InputFile{fileLocation, separator, pretty}, nil
}

func IsValidFile(filename string) (bool, error) {
	if fileExt := filepath.Ext(filename); fileExt != ".csv" {
		return false, fmt.Errorf("file %s is not CSV", filename)
	}

	if _, err := os.Stat(filename); err != nil && os.IsNotExist(err) {
		return false, fmt.Errorf("file %s does not exist", filename)
	}

	return true, nil
}

func ProcessCSVFile(fileData InputFile, writerChan chan<- map[string]interface{}) {
	file, err := os.Open(fileData.Filepath)
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
			ExitGracefully(err)
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
		ExitGracefully(e)
	}
}

func ExitGracefully(err error) {
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

func WriteJSONFile(csvPath string, writerChannel <-chan map[string]interface{}, done chan<- bool, pretty bool) {
	writeString := createStringWriter(csvPath, pretty)
	jsonFunc, breakLine := getJSONFunc(pretty)

	fmt.Println("Writing JSON file...")

	writeString("["+breakLine, false)
	first := true
	for {
		// Waiting for pushed records into our writerChannel
		record, more := <-writerChannel
		if more {
			if !first {
				writeString(","+breakLine, false)
			} else {
				first = false
			}

			jsonData := jsonFunc(record)
			writeString(jsonData, false)
		} else {
			writeString(breakLine+"]", true)
			fmt.Println("Completed!")
			done <- true
			break
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

	var jsonFunc func(map[string]interface{}) string
	var breakLine string
	if pretty {
		breakLine = "\n"
		jsonFunc = func(record map[string]interface{}) string {
			jsonData, _ := json.MarshalIndent(record, "   ", "   ")
			return "   " + string(jsonData)
		}
	} else {
		breakLine = ""
		jsonFunc = func(record map[string]interface{}) string {
			jsonData, _ := json.Marshal(record)
			return string(jsonData)
		}
	}

	return jsonFunc, breakLine
}
