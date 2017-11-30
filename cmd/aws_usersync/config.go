package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/UKHomeOffice/aws_usersync/pkg/log"
)

func checkGroups() {
	if *group == "" {
		log.Fatal("Group is not specified. You must specify a group in IAM that contains your users")
		os.Exit(1)
	}
}

func printVersion() {
	if *versionShow {
		fmt.Printf("%s: %s\n", binName, version)
		os.Exit(0)
	}
}

func checkOptions() {
	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(0)
	} else {
		printVersion()
		checkGroups()
		if *logLevel != "" {
			log.SetLevel(*logLevel)
		}
	}
}
func flagOptions() {
	flag.Parse()
	checkOptions()
}
