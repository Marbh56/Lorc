package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var filename string

	flag.StringVar(&filename, "file", "", "JSON file to validate")
	flag.Parse()

	if filename == "" {
		fmt.Println("Please provide a file with -file flag")
		return
	}

	valid, err := validJSON(filename)
	if err != nil {
		fmt.Printf("Error processing file: %v\n", err)
		return
	}
	if valid {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func validJSON(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return false, err
	}

	size := fileInfo.Size()

	if size == 0 {
		return false, nil
	}

	firstChar := make([]byte, 1)
	_, err = file.Read(firstChar)
	if err != nil {
		return false, err
	}

	if firstChar[0] != '{' {
		return false, nil
	}

	_, err = file.Seek(size-1, 0)
	if err != nil {
		return false, err
	}

	lastChar := make([]byte, 1)
	_, err = file.Read(lastChar)
	if err != nil {
		return false, err
	}
	if lastChar[0] != '}' {
		return false, nil
	}

	return true, nil

}
