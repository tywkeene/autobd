package index

import (
	"testing"
)

func TestGetChecksumSuccess(t *testing.T) {
	var fileName string = "index.go"
	var checksum string = ""

	checksum, err := GetChecksum(fileName)
	if err != nil {
		t.Error(err)
	}
	if checksum == "" {
		t.Errorf("Failed to get checksum for index.go")
	}
}

func TestGetChecksumInvalidFileName(t *testing.T) {
	var fileName string = "index.no"
	var checksum string = ""
	checksum, err := GetChecksum(fileName)
	if err == nil {
		t.Errorf("Error should not be nil")
	}
	if checksum != "" {
		t.Errorf("Somehow got checksum invalid file %s: %s\n", fileName, checksum)
	}
}

func TestGetIndex(t *testing.T) {
	var dirPath string = "./"

	returnedIndex, err := GetIndex(dirPath)

	if returnedIndex == nil {
		t.Errorf("Index should not be nil")
	}

	if err != nil {
		t.Error(err)
	}
}
