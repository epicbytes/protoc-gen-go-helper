package main

import (
	"fmt"
	"regexp"
	"testing"
)

func TestParseMerge(t *testing.T) {
	line := "message AdminJetonUpdateRequest {// @parser:\"fiber\",@parser:\"swag\",@merge:\"AdminJetonUpdateRequest|JetonEntity\""
	re := regexp.MustCompile(`(?m)@merge:"(.*)\|(.*)"`)
	for _, match := range re.FindAllStringSubmatch(line, -1) {
		fmt.Println(match[2])
	}
}
