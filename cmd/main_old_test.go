package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func Test_getFileData(t *testing.T) {
	tests := []struct {
		name    string
		want    inputFile
		wantErr bool
		osArgs  []string
	}{
		{"Default parameters", inputFile{"test.csv", "comma", false}, false, []string{"cmd", "test.csv"}},
		{"No parameters", inputFile{}, true, []string{"cmd"}},
		{"Semicolon enabled", inputFile{"test.csv", "semicolon", false}, false, []string{"cmd", "--separator=semicolon", "test.csv"}},
		{"Pretty enabled", inputFile{"test.csv", "comma", true}, false, []string{"cmd", "--pretty", "test.csv"}},
		{"Pretty and semicolon enabled", inputFile{"test.csv", "semicolon", true}, false, []string{"cmd", "--pretty", "--separator=semicolon", "test.csv"}},
		{"Separator not identified", inputFile{}, true, []string{"cmd", "--separator=pipe", "test.csv"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			actualOsArgs := os.Args

			defer func() {
				os.Args = actualOsArgs                                           // Restoring the original os.Args reference
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError) // Reset the Flag command line. So that we can parse flags again
			}()

			os.Args = tt.osArgs
			got, err := getFileData()
			if (err != nil) != tt.wantErr {
				t.Errorf("getFileData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getFileData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isValidFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test*.csv")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	tests := []struct {
		name     string
		filename string
		want     bool
		wantErr  bool
	}{
		{"File does exist", tmpfile.Name(), true, false},
		{"File does not exist", "nowhere/test.csv", false, true},
		{"File is not csv", "test.txt", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isValidFile(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkIfValidFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("checkIfValidFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processCSVFile(t *testing.T) {
	wantMapSlice := []map[string]string{
		{"COL1": "1", "COL2": "2", "COL3": "3"},
		{"COL1": "4", "COL2": "5", "COL3": "6"},
	}

	tests := []struct {
		name      string
		csvString string
		separator string
	}{
		{"Comma separator", "COL1,COL2,COL3\n1,2,3\n4,5,6\n", "comma"},
		{"Semicolon separator", "COL1;COL2;COL3\n1;2;3\n4;5;6\n", "semicolon"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Creating a CSV temp file for testing
			dumpfile, err := os.CreateTemp("", "test*.csv")
			check(err)
			defer os.Remove(dumpfile.Name())

			_, err = dumpfile.WriteString(tt.csvString)
			if err != nil {
				t.Errorf("writing to dumpfile failed: %e", err)
			}

			err = dumpfile.Sync()
			if err != nil {
				t.Errorf("sync dumpfile failed: %e", err)
			}

			testFileData := inputFile{
				filepath:  dumpfile.Name(),
				pretty:    false,
				separator: tt.separator,
			}

			writerChannel := make(chan map[string]string)

			go processCSVFile(testFileData, writerChannel)

			for _, wantMap := range wantMapSlice {
				record := <-writerChannel
				fmt.Println()
				fmt.Println(wantMap)
				fmt.Println(record)
				fmt.Println()
				if !reflect.DeepEqual(record, wantMap) {
					t.Errorf("processCsvFile() = %v, want %v", record, wantMap)
				}
			}
		})
	}
}

func Test_writeJSONFile(t *testing.T) {
	dataMap := []map[string]string{
		{"COL1": "1", "COL2": "2", "COL3": "3"},
		{"COL1": "4", "COL2": "5", "COL3": "6"},
	}

	tests := []struct {
		csvPath  string
		jsonPath string
		pretty   bool
		name     string
	}{
		{"compact.csv", "compact.json", false, "Compact JSON"},
		{"pretty.csv", "pretty.json", true, "Pretty JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			writerChannel := make(chan map[string]string)
			done := make(chan bool)

			go func() {
				for _, record := range dataMap {
					writerChannel <- record
				}
				close(writerChannel)
			}()

			go writeJSONFile(tt.csvPath, writerChannel, done, tt.pretty)

			<-done

			testOutput, err := os.ReadFile(tt.jsonPath)
			if err != nil {
				t.Errorf("writeJSONFile(), Output file got error: %v", err)
			}
			defer os.Remove(tt.jsonPath)

			wantOutput, err := os.ReadFile(filepath.Join("../testJsonFiles", tt.jsonPath))
			check(err)

			if !bytes.Equal(testOutput, wantOutput) {
				t.Errorf("\nwriteJSONFile() = \n%v \nwant = \n%v", string(testOutput), string(wantOutput))
			}
		})
	}
}
