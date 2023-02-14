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

func canBeIgnored(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 ||
		strings.HasPrefix(line, "#") ||
		strings.HasPrefix(line, ";") {
		return true
	}
	return false
}

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

func (s sectionList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s sectionList) Len() int           { return len(s) }
func (s sectionList) Less(i, j int) bool { return s[i].Order < s[j].Order }

func save(config *SambaConfig) error {
	// make slice
	sections := make(sectionList, 0, len(config.Sections))
	for _, section := range config.Sections {
		sections = append(sections, section)
	}
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

	// TODO file validation

	// rename file
	if err := os.Rename("smb.conf.temp", config.FilePath); err != nil {
		return err
	}
	return nil
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

func (sc *SambaConfig) UpdSection(section *Section) error {
	// check exist
	if _, ok := sc.Sections[section.Name]; !ok {
		return fmt.Errorf("section '%s' does not exist", section.Name)
	}
	// check params exist
	if len(section.Params) == 0 {
		return errors.New("params cannot be empty")
	}
	// update section
	sc.Sections[section.Name] = section
	return save(sc)
}

func (sc *SambaConfig) GetSection(sname string) (*Section, error) {
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
	section, ok := sc.Sections[sname]
	if !ok {
		return fmt.Errorf("section '%s' does not exist", sname)
	}
	param := &Param{Key: key, Value: value}
	section.Params[len(section.Params)] = param
	return save(sc)
}

func (sc *SambaConfig) DelParamByKey(sname, key string) error {
	section, ok := sc.Sections[sname]
	if !ok {
		return fmt.Errorf("section '%s' does not exist", sname)
	}
	var keyExist bool
	for index, param := range section.Params {
		if param.Key == key {
			section.Params = append(section.Params[:index], section.Params[index+1:]...)
			keyExist = true
		}
	}
	if !keyExist {
		return fmt.Errorf("param '%s' in section '%s' does not exist", key, sname)
	}
	return save(sc)
}
