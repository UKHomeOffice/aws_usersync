package main

import (
 "fmt"
 "os"
 "flag"
 "github.com/UKHomeOffice/aws_usersync/log"
)

func checkGroups() {
  if *groups == "" {
    log.Fatal("Groups are not specified. You must specify a list of comma separated groups")
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
