package loader

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"testing"
)

// go test -run LoadXML -v
func TestLoadXML(t *testing.T) {

	// 打开文件
	inputFile, inputError := os.Open("D:\\Temp\\20180612001616173434.xml")

	if inputError != nil {
		fmt.Println("error in open file.")
		t.FailNow()
	}

	_, err := LoadXML(xml.NewDecoder(bufio.NewReader(inputFile)))

	if err != nil {
		t.FailNow()
	}

	inputFile.Close()
}

// 执行go test -bench LoadXML -v
func BenchmarkLoadXML(b *testing.B) {

	// 打开文件
	inputFile, inputError := os.Open("D:\\Temp\\20180612001616173434.xml")

	if inputError != nil {
		fmt.Println("error in open file.")
		b.FailNow()
	}

	_, err := LoadXML(xml.NewDecoder(bufio.NewReader(inputFile)))

	if err != nil {
		b.Fail()
	}

	inputFile.Close()
}
