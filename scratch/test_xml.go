package main

import (
	"encoding/xml"
	"fmt"
)

type XMLGeneralInformation struct {
	XMLName xml.Name `xml:"applejuice"`
	General struct {
		Version string `xml:"version"`
		System  string `xml:"system"`
	} `xml:"generalinformation"`
}

func main() {
	data := `
<applejuice>
<generalinformation>
<version>0.34.101.42</version>
<filesystem seperator="/"/>
<system>Linux</system>
</generalinformation>
</applejuice>`

	var info XMLGeneralInformation
	err := xml.Unmarshal([]byte(data), &info)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Version: '%s'\n", info.General.Version)
	fmt.Printf("System: '%s'\n", info.General.System)
}
