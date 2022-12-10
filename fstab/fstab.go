package fstab

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
)

const DefaultFstabPath = "/etc/fstab"

type Identifior struct {
	Key   string
	Value string
}

type Record struct {
	Device     *Identifior
	Mountpoint string
	FsType     string
	Options    string
	Dump       int
	Pass       int
	RawLineNo  int
}

type Fstab struct {
	sync.Mutex
	RawLines []string
	Records  map[int]*Record
}

// remove extra sapaces
func removeExtraSpaces(arr []byte) string {

	if len(arr) == 0 {
		return ""
	}

	tempArr := make([]byte, 0, len(arr))
	var preChar byte

	for _, curChar := range arr {
		if curChar == 32 && preChar == 32 {
			continue
		}
		tempArr = append(tempArr, curChar)
		preChar = curChar
	}
	return string(tempArr)
}

func parseIdentifior(item string) *Identifior {

	if len(item) == 0 {
		return nil
	}

	identifior := &Identifior{}

	if !strings.Contains(item, "=") {
		identifior.Key = ""
		identifior.Value = item
		return identifior
	}

	kv := strings.Split(item, "=")
	identifior.Key = kv[0]
	identifior.Value = kv[1]

	return identifior
}

func (fs *Fstab) Load() error {
	fs.Lock()
	defer fs.Unlock()

	// read fstab file
	raw, err := ioutil.ReadFile(DefaultFstabPath)
	if err != nil {
		return err
	}

	content := string(raw)
	lines := strings.Split(content, "\n")

	// initialize record slice
	records := make(map[int]*Record)

	for idx, line := range lines {
		// trime spaces
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// replace tab for a sign space
		line = strings.ReplaceAll(line, "\t", " ")

		// ignore commented line
		if line[0:1] == "#" {
			continue
		}

		// remove extra spaces
		line = removeExtraSpaces([]byte(line))
		items := strings.Split(line, " ")
		if len(items) != 6 {
			continue
		}

		// set record values
		record := &Record{
			Device:     parseIdentifior(items[0]),
			Mountpoint: items[1],
			FsType:     items[2],
			Options:    items[3],
			RawLineNo:  idx,
		}
		dump, _ := strconv.Atoi(items[4])
		pass, _ := strconv.Atoi(items[5])
		record.Dump = dump
		record.Pass = pass

		// add to record slice
		records[idx] = record
	}

	// set raw content lines
	fs.RawLines = lines
	// set record slice
	fs.Records = records

	return nil
}

func (fs *Fstab) GetRecordByMountpoint(mp string) (Record, error) {
	fs.Lock()
	defer fs.Unlock()

	for _, r := range fs.Records {
		if mp == r.Mountpoint {
			return *r, nil
		}
	}
	return Record{}, fmt.Errorf("the specified mountpoint record was not found")
}

// ReplaceContentByMountpoint replace the content of specified record
func (fs *Fstab) ReplaceContentByMountpoint(record *Record) error {

	fs.Lock()
	defer fs.Unlock()

	var (
		rawLineNo int
	)

	for _, r := range fs.Records {
		if record.Mountpoint == r.Mountpoint {
			rawLineNo = r.RawLineNo
			break
		}
	}

	var deviceDescriptor string
	if len(record.Device.Key) == 0 {
		deviceDescriptor = record.Device.Value
	} else {
		deviceDescriptor = record.Device.Key + "=" + record.Device.Value
	}

	items := []string{
		deviceDescriptor,
		record.Mountpoint,
		record.FsType,
		record.Options,
		strconv.Itoa(record.Dump),
		strconv.Itoa(record.Pass),
	}
	// use temp parameters
	tempLines := make([]string, len(fs.RawLines))
	copy(tempLines, fs.RawLines)
	tempLines[rawLineNo] = strings.Join(items, " ")
	content := strings.Join(tempLines, "\n")

	if err := ioutil.WriteFile(DefaultFstabPath, []byte(content), 0644); err != nil {
		return err
	}

	// after the file successfully written, save the newest value in rom
	fs.Records[rawLineNo] = record
	fs.RawLines[rawLineNo] = tempLines[rawLineNo]

	return nil
}
