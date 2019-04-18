package main

import "testing"

func TestValidateHTTPProps(t *testing.T) {
	props := make(map[string]string)
	props["url"] = "http://nowhere"
	props["method"] = "GET"

	check := Checker{"", "", props}
	err := validateHTTPProps(check)
	if err != nil {
		t.Errorf("Should not return error if properties exist")
	}

}

func TestValidateHTTPPropsMissing(t *testing.T) {
	props := make(map[string]string)
	//props["url"] = "http://nowhere" //Missing URL
	props["method"] = "GET"

	check := Checker{"", "", props}
	err := validateHTTPProps(check)
	if err == nil {
		t.Errorf("Should return error if properties is missing")
	}

}
