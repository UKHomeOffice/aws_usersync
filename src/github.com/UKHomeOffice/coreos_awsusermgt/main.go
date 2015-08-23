package main

import (
  "fmt"
	"flag"
  "strings"
  "os"
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/service/iam"

  ciam "github.com/UKHomeOffice/coreos_awsusermgt/coreos_iam"
)

const (
	version = "0.0.1"
)

// custom type for maps to userData
type userMap map[string]*userData

// structure to hold the user data for users
type userData struct {
	group string
  keys  []string
}


// Define variables flag and standard
var (
  keyEncoding   = flag.String("e", "SSH", "SSH Key encoding type ssh-rsa or pem, defaults to SSH")
  sudoGroup     = flag.String("S", "sudo", "Group for the users to be part of for sudo, defaults to sudo group")
  groups        = flag.String("g", "", "Comma separated list of Group names in AWS")
  versionShow   = flag.Bool("v", false, "Display the version")
  region        = flag.String("r", "eu-west-1", "AWS Region, defaults to eu-west-1")
  binName       = "coreos_awsusermgt"
)

// wrapper function for stderr
func stderr(f string, a ...interface{}) {
  out := fmt.Sprintf(f, a...)
  fmt.Fprintln(os.Stderr, strings.TrimSuffix(out, "\n"))
}

// wrapper function for stdout
func stdout(f string, a ...interface{}) {
  out := fmt.Sprintf(f, a...)
  fmt.Fprintln(os.Stdout, strings.TrimSuffix(out, "\n"))
}

// wrapper function for panic
func panicf(f string, a ...interface{}) {
  panic(fmt.Sprintf(f, a...))
}

func splitGroups(g string) []string {
  glist := strings.Split(strings.Replace(g, " ", "", -1), ",")
  return glist
}

func (u userMap) setKey(svc *iam.IAM) {
  for user, struc := range u {
    keys, err := ciam.GetKeys(user, svc)
    if err != nil {
      stderr("Error occurred getting keys: %v", err)
    }
    if len(keys) == 0 {
      stdout("No active keys for \"%v\". Not adding user (get them to add their key)\n", user)
      delete(u, user)
    } else {
      struc.keys = keys
    }
  }
}

func (u userMap) setIamUsers(svc *iam.IAM, g []string) {
  for _, grp := range g {
    resp, err := svc.GetGroup(&iam.GetGroupInput{GroupName: aws.String(grp)})
    if err != nil {
      stderr("Error getting Group: %v, %v", grp, err)
    }
    for _, user := range ciam.GetIamUsers(resp) {
      u[user] = &userData{group: grp}
    }
  }
}

func (u userMap) printMap() {
  for user, struc := range u {
    fmt.Printf("\nUser: %v, Data: %+v\n", user, struc)
  }
}

// function main call out into validate code
func main() {
	flagOptions()
  grpList := splitGroups(*groups)

  // send configuration to aws and then get the svc reference
  cfg := &aws.Config{Region: aws.String(*region)}
  iamsvc := ciam.GetIamClient(cfg)

  // Make and initaize the map for structure
  umap := make(userMap)

  // Fetch all iam users from group and assign to userMap type
  umap.setIamUsers(iamsvc, grpList)

  // Set all the keys for users
  umap.setKey(iamsvc)

  // Print data structure
  umap.printMap()

}
