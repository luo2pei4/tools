package smbconf

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
)

type Param struct {
	Key   string
	Value string
}

type Section struct {
	Name   string
	Order  int
	Params []*Param
}

type SambaConfig struct {
	FilePath string
	Sections map[string]*Section
}

const DefaultSmbConfPath = "/etc/samba/smb.conf"

var (
	sectionReg = regexp.MustCompile(`\[.+]`)
	paramReg   = regexp.MustCompile(`\s*.+=.*`)
)

func LoadConfigFile(path string) (*SambaConfig, error) {
	// open smb.conf file
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var (
		section      *Section
		sectionName  string
		sectionOrder int
		sambaConfig  = &SambaConfig{
			FilePath: path,
			Sections: make(map[string]*Section),
		}
	)

	// scan file
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// check commented or blank line
		if len(line) == 0 ||
			strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, ";") {
			continue
		}
		// section
		if sectionReg.MatchString(line) {
			sectionName = line[1 : len(line)-1]
			section = &Section{
				Name:   sectionName,
				Order:  sectionOrder,
				Params: make([]*Param, 0),
			}
			sambaConfig.Sections[sectionName] = section
			sectionOrder++
		}
		// section
		if paramReg.MatchString(line) {
			kv := strings.Split(line, "=")
			param := &Param{
				Key:   strings.TrimSpace(kv[0]),
				Value: strings.TrimSpace(kv[1]),
			}
			section.Params = append(section.Params, param)
		}
	}
	return sambaConfig, nil
}

type sectionList []*Section

// implements interface sort.Interface
func (s sectionList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sectionList) Len() int           { return len(s) }
func (s sectionList) Less(i, j int) bool { return s[i].Order < s[j].Order }

// save smb.conf file
func save(config *SambaConfig) error {
	// make slice
	sections := make(sectionList, 0, len(config.Sections))
	for _, section := range config.Sections {
		sections = append(sections, section)
	}
	// sort by section order
	sort.Sort(sections)

	var builder strings.Builder
	for _, section := range sections {
		builder.WriteString("[" + section.Name + "]")
		builder.WriteRune('\n')
		for _, param := range section.Params {
			if param == nil {
				continue
			}
			builder.WriteString("    " + param.Key + " = " + param.Value)
			builder.WriteRune('\n')
		}
		builder.WriteRune('\n')
	}

	// write temp file
	if err := ioutil.WriteFile("smb.conf.temp", []byte(builder.String()), 0644); err != nil {
		return err
	}

	// TODO file validation, call testparm

	// rename file
	return os.Rename("smb.conf.temp", config.FilePath)
}

// ################################# section operations #################################

func (sc *SambaConfig) AddSection(section *Section) error {
	// check exist
	if _, ok := sc.Sections[section.Name]; ok {
		return fmt.Errorf("section '%s' has already exist", section.Name)
	}
	// check params exist
	if len(section.Params) == 0 {
		return errors.New("params cannot be empty")
	}
	// add section
	sc.Sections[section.Name] = section
	return save(sc)
}

func (sc *SambaConfig) DelSection(sname string) error {
	// check exist
	if _, ok := sc.Sections[sname]; !ok {
		return fmt.Errorf("section '%s' does not exist", sname)
	}
	// delete section
	delete(sc.Sections, sname)
	return save(sc)
}

func getSection(sc *SambaConfig, sname string) (*Section, error) {
	if len(strings.TrimSpace(sname)) == 0 {
		return nil, fmt.Errorf("invalid parameter, section name: '%s'", sname)
	}
	// check exist
	section, ok := sc.Sections[sname]
	if !ok {
		return nil, fmt.Errorf("section '%s' does not exist", sname)
	}
	return section, nil
}

// ################################# param operations #################################

func (sc *SambaConfig) AddParam(sname, key, value string) error {
	section, err := getSection(sc, sname)
	if err != nil {
		return err
	}
	param := &Param{Key: key, Value: value}
	section.Params = append(section.Params, param)
	return save(sc)
}

func (sc *SambaConfig) DelParamByKey(sname, key string) error {
	section, err := getSection(sc, sname)
	if err != nil {
		return err
	}
	var keyExist bool
	params := make([]*Param, 0, len(section.Params))
	for _, param := range section.Params {
		if param.Key == key {
			keyExist = true
			continue
		}
		params = append(params, param)
	}
	section.Params = params
	if !keyExist {
		return fmt.Errorf("param '%s' in section '%s' does not exist", key, sname)
	}
	return save(sc)
}

func (sc *SambaConfig) DelParamByKeyAndValue(sname, key, value string) error {
	section, err := getSection(sc, sname)
	if err != nil {
		return err
	}
	var keyExist bool
	params := make([]*Param, 0, len(section.Params))
	for _, param := range section.Params {
		if param.Key == key && param.Value == value {
			keyExist = true
			continue
		}
		params = append(params, param)
	}
	section.Params = params
	if !keyExist {
		return fmt.Errorf("param '%s' with value '%s' in section '%s' does not exist", key, value, sname)
	}
	return save(sc)
}

func (sc *SambaConfig) UpdParamByKey(sname, key, value string) error {
	section, err := getSection(sc, sname)
	if err != nil {
		return err
	}
	var keyExist bool
	for _, param := range section.Params {
		if param.Key == key {
			param.Value = value
			keyExist = true
		}
	}
	if !keyExist {
		return fmt.Errorf("param '%s' in section '%s' does not exist", key, sname)
	}
	return save(sc)
}

func (sc *SambaConfig) UpdParamByKeyAndValue(sname, key, oldValue, newValue string) error {
	section, err := getSection(sc, sname)
	if err != nil {
		return err
	}
	var keyExist bool
	for _, param := range section.Params {
		if param.Key == key && param.Value == oldValue {
			param.Value = newValue
			keyExist = true
		}
	}
	if !keyExist {
		return fmt.Errorf("param '%s' with value '%s' in section '%s' does not exist", key, oldValue, sname)
	}
	return save(sc)
}
