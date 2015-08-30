package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/UKHomeOffice/coreos_awsusermgt/coreos_iam"
	"github.com/UKHomeOffice/coreos_awsusermgt/coreos_users"
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
	keyEncoding = flag.String("e", "SSH", "SSH Key encoding type ssh-rsa or pem, defaults to SSH")
	sudoGroup   = flag.String("S", "sudo", "Group for the users to be part of for sudo, defaults to sudo group")
	groups      = flag.String("g", "", "Comma separated list of Group names in AWS")
	versionShow = flag.Bool("v", false, "Display the version")
	region      = flag.String("r", "eu-west-1", "AWS Region, defaults to eu-west-1")
	binName     = "coreos_awsusermgt"
	grpList     []string
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

// Split the group list into an array
func splitGroups(g string) []string {
	glist := strings.Split(strings.Replace(g, " ", "", -1), ",")
	return glist
}

// check if string is in a slice array
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
  	if b == a {
    	return true
    }
  }
  return false
}

// Set the key in the structure for the user fetched from iam or delete user from
// structure if the user hasn't set a key
func (u userMap) setKey(svc *iam.IAM) {
	for user, struc := range u {
		keys, err := coreos_iam.GetKeys(user, svc)
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

// Set the IAM users
func (u userMap) setIamUsers(svc *iam.IAM, g []string) {
	for _, grp := range g {
		resp, err := svc.GetGroup(&iam.GetGroupInput{GroupName: aws.String(grp)})
		if err != nil {
			stderr("Error getting Group: %v, %v", grp, err)
		}
		for _, user := range coreos_iam.GetIamUsers(resp) {
			u[user] = &userData{group: grp}
		}
	}
}

func (u userMap) printMap() {
	for user, struc := range u {
		fmt.Printf("\nUser: %v, Data: %+v\n", user, struc)
	}
}

func addUser(usrStr string, group string, sgroup string, keys []string) {
	luser := coreos_users.New(usrStr, group, sgroup, keys)
	out, err := luser.Sync()
	if err != nil {
		stderr("Error syncing users: %v", err)
	}
	stdout("User Info ..... :: %v", out)
}

// Return the difference between iam users and local users as array
func (u userMap) diffUsers() ([]string) {
	var iamusers []string
	localusers, err := coreos_users.GetAllUsers()
	stdout("Localusers: %v", localusers)
	if err != nil {
		stderr("An error occured grabbing users from system: %v", err)
	}
	for user, _ := range u {
		iamusers = append(iamusers, user)
	}
	diffusers := coreos_users.GetArrayDiff(iamusers, localusers)
	return diffusers
}


// Loop through all the users
func (u userMap) loopUsers() {
	for userStr, data := range u {
		if _, err := user.Lookup(userStr); err != nil {
			addUser(userStr, data.group, *sudoGroup, data.keys)
		}
		for _, usr := range u.diffUsers() {
			ignoreUsers := []string{"root", "core"}
			if stringInSlice(usr, ignoreUsers) {
				continue
			}
			stdout("Removing local user: %v as not in the Group List", usr)
			if err := coreos_users.RemoveUser(usr); err != nil {
				stderr("Error removing user: %v", err)
			}
		}
	}
}

// function main call out into validate code
func main() {
	flagOptions()
	grpList = splitGroups(*groups)

	// send configuration to aws and then get the svc reference
	cfg := &aws.Config{Region: aws.String(*region)}
	iamsvc := coreos_iam.GetIamClient(cfg)

	// Make and initaize the map for structure
	umap := make(userMap)

	// Fetch all iam users from group and assign to userMap type
	umap.setIamUsers(iamsvc, grpList)

	// Set all the keys for users
	umap.setKey(iamsvc)

	// set users
	umap.loopUsers()

	// Print data structure
//	umap.printMap()

}
