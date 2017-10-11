package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/uswitch/aws_usersync/log"
	"github.com/uswitch/aws_usersync/sync_iam"
	"github.com/uswitch/aws_usersync/sync_users"
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
	interval    = flag.Int("i", 30, "The frequency to poll in Minutes, for updates from the cloud provider")
	ignoreusers = flag.String("I", "root,core", "Specify comma separated list of users to ignore on the system so they wont be attempted to be removed")
	onetime     = flag.Bool("o", true, "One time run as oppose polling and daemonizing")
	logLevel    = flag.String("L", "", "Set the log level: Error, Info, Debug")
	region      = flag.String("r", "eu-west-1", "AWS Region, defaults to eu-west-1")
	binName     = "coreos_awsusermgt"
	grpList     []string
)

// Split the group list into an array
func splitString(g string) []string {
	glist := strings.Split(strings.Replace(g, " ", "", -1), ",")
	return glist
}

// Set the key in the structure for the user fetched from iam or delete user from
// structure if the user hasn't set a key
func (u userMap) setKey(svc *iam.IAM) error {
	for user, struc := range u {
		keys, err := sync_iam.GetKeys(user, svc)
		if err != nil {
			log.Error(fmt.Sprintf("Error occurred getting keys: %v", err))
			return err
		}
		if len(keys) == 0 {
			log.Debug(fmt.Sprintf("No active keys for %v. Not adding user [get them to add their key]", user))
			delete(u, user)
		} else {
			struc.keys = keys
		}
	}
	return nil
}

// Set the IAM users
func (u userMap) setIamUsers(svc *iam.IAM, g []string) error {
	for _, grp := range g {
		resp, err := svc.GetGroup(&iam.GetGroupInput{GroupName: aws.String(grp)})
		if err != nil {
			log.Error(fmt.Sprintf("Error getting Group: %v, %v", grp, err))
			return err
		}
		for _, user := range sync_iam.GetIamUsers(resp) {
			u[user] = &userData{group: grp}
		}
	}
	return nil
}

// Take in the channels and loop continuously through until we need to break
// syncing uses with whatever the time interval supplied is
func (u userMap) process(grp []string, doneChan chan bool, stopChan chan bool, interval int) {
	defer close(doneChan)
	for {
		u.userSync(grp)
		select {
		case <-stopChan:
			break
		case <-time.After(time.Duration(interval) * time.Minute):
			continue
		}
	}
}

// Loop through all the users and add user locally to main which will call sync
// we need to call dokeys on if the user exists or not so need to check the error message
// for whether the user exists or not and run it anyway unless some other error
func (u userMap) userSync(grp []string) error {
	// set region
	cfg := &aws.Config{Region: aws.String(*region)}
	// set Iam service
	iamsvc := sync_iam.GetIamClient(cfg)

	// Fetch all iam users from group and assign to userMap type
	if err := u.setIamUsers(iamsvc, grp); err != nil {
		return err
	}

	// Set all the keys for users
	if err := u.setKey(iamsvc); err != nil {
		return err
	}
	var IamUsers []string
	for userStr, data := range u {
		IamUsers = append(IamUsers, userStr)
		luser := sync_users.New(userStr, data.group, *sudoGroup, data.keys)
		if err := luser.Sync(); err != nil {
			log.Error(fmt.Sprintf("Error syncing users: %v", err))
			return err
		}
	}
	ignored := splitString(*ignoreusers)
	userCmp, err := sync_users.CmpNew(IamUsers, ignored)
	if err != nil {
		return err
	}
	if err := userCmp.Cleanup(); err != nil {
		return err
	}
	return nil
}

// function main call out into validate code
func main() {
	// Make and initaize the map for structure
	umap := make(userMap)

	// Check the flag options
	flagOptions()

	// Get a list of the groups
	grpList = splitString(*groups)

	if *onetime {
		if err := umap.userSync(grpList); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Set the channels
	stopChan := make(chan bool)
	doneChan := make(chan bool)
	errChan := make(chan error, 10)

	go umap.process(grpList, doneChan, stopChan, *interval)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case err := <-errChan:
			log.Error(fmt.Sprintf("Error captured: %v", err.Error()))
		case s := <-signalChan:
			log.Info(fmt.Sprintf("Captured %v. Exiting...", s))
			close(doneChan)
		case <-doneChan:
			os.Exit(0)
		}
	}

}
