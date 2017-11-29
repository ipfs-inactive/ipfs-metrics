package main

import (
	"fmt"
	"testing"
)

func TestValidTags(t *testing.T) {
	validTags := [...]string{"t1=v1", "t_2=v_2", "t3=3", "@=@", `"tater"="tot"`, "123=456", "'tomato'='tomoto'"}
	for tag := range validTags {
		if !ValidTag(validTags[tag]) {
			t.Error(fmt.Sprintf("Tag is valid: %s", validTags[tag]))
		}
	}

}

func TestInValidTags(t *testing.T) {
	invalidTags := [...]string{"t1", "=", "t2=v2=vv2", ",tag=val", "val=,tag", "tag,=val", "tag=val,", "===", `test=prefix\`}
	for tag := range invalidTags {
		if ValidTag(invalidTags[tag]) {
			t.Error(fmt.Sprintf("Tag is invalid: %s", invalidTags[tag]))
		}
	}
}

func TestMakeTags(t *testing.T) {
	tagsToMake := []string{"t1=v1", "t_2=v_2", "t3=3", "@=@", "123=456", "'tomato'='tomoto'"}
	var madeTags = []Tag{
		{
			Name:  "t1",
			Value: "v1",
		},
		{
			Name:  "t_2",
			Value: "v_2",
		},
		{
			Name:  "t3",
			Value: "3",
		},
		{
			Name:  "@",
			Value: "@",
		},
		{
			Name:  "123",
			Value: "456",
		},
		{
			Name:  "'tomato'",
			Value: "'tomoto'",
		},
	}
	maybeTags, err := MakeTags(tagsToMake)
	if err != nil {
		t.Error("Failed To Make Tags For Test")
	}
	for mt := range madeTags {
		if maybeTags[mt] != madeTags[mt] {
			t.Error(fmt.Sprintf("Invalid Tag: %v Expected: %v", maybeTags[mt], madeTags[mt]))
		}
	}
}
