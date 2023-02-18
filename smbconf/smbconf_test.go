package smbconf

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLoadConfigFile(t *testing.T) {

	// load file
	conf, err := LoadConfigFile("smb.conf")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	arr, err := json.Marshal(conf)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(arr))

	// add section
	section := &Section{
		Name:  "test",
		Order: len(conf.Sections),
		Params: []*Param{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: "value2"},
			{Key: "key3", Value: "value3"},
			{Key: "key4", Value: "value4"},
			{Key: "key5", Value: "value5"},
			{Key: "key6", Value: "value6"},
			{Key: "key6", Value: "value6-1"},
		},
	}
	err = conf.AddSection(section)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// delete section
	err = conf.DelSection("test")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// add section
	section = &Section{
		Name:  "test",
		Order: len(conf.Sections),
		Params: []*Param{
			{Key: "key1", Value: "value1"},
			{Key: "key2", Value: "value2"},
			{Key: "key3", Value: "value3"},
			{Key: "key3", Value: "value3-1"},
			{Key: "key4", Value: "value4"},
			{Key: "key5", Value: "value5"},
			{Key: "key5", Value: "value5-1"},
			{Key: "key6", Value: "value6"},
			{Key: "key6", Value: "value6-1"},
		},
	}
	err = conf.AddSection(section)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// AddParam
	err = conf.AddParam("test", "key7", "value7")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// DelParamByKey
	err = conf.DelParamByKey("test", "key5")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// DelParamByKeyAndValue
	err = conf.DelParamByKeyAndValue("test", "key6", "value6-1")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// UpdParamByKey
	err = conf.UpdParamByKey("test", "key2", "value2-1")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// UpdParamByKeyAndValue
	err = conf.UpdParamByKeyAndValue("test", "key3", "value3-1", "value3-updated")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
