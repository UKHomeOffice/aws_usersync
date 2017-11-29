package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/UKHomeOffice/aws_usersync/pkg/log"
	iam "github.com/UKHomeOffice/aws_usersync/pkg/sync_iam"
	"github.com/UKHomeOffice/aws_usersync/pkg/sync_users"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	version = "0.1.1"
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
	group       = flag.String("g", "", "The group in AWS that contains the users")
	versionShow = flag.Bool("v", false, "Display the version")
	interval    = flag.Int("i", 30, "The frequency to poll in Minutes, for updates from the cloud provider")
	ignoreusers = flag.String("I", "root,core", "Specify comma separated list of users to ignore on the system so they wont be attempted to be removed")
	onetime     = flag.Bool("o", true, "One time run as oppose polling and daemonizing")
	logLevel    = flag.String("L", "", "Set the log level: Error, Info, Debug")
	region      = flag.String("r", "eu-west-1", "AWS Region, defaults to eu-west-1")
	binName     = "coreos_awsusermgt"
	grpList     []string
)

func init() {
	// set region
	cfg := &aws.Config{Region: aws.String(*region)}
	sess := session.Must(session.NewSession())
	// set Iam service
	iam.NewIAM(sess, cfg)
}

// Split the group list into an array
func splitString(g string) []string {
	glist := strings.Split(strings.Replace(g, " ", "", -1), ",")
	return glist
}

// Set the key in the structure for the user fetched from iam or delete user from
// structure if the user hasn't set a key
func (u userMap) setKey() error {
	for user, umap := range u {
		keys, err := iam.IAMsvc.GetKeys(user)
		if err != nil {
			log.Error(fmt.Sprintf("Error occurred getting keys: %v", err))
			return err
		}
		if len(keys) == 0 {
			log.Debug(fmt.Sprintf("No active keys for %v. Not adding user [get them to add their key]", user))
			delete(u, user)
		} else {
			umap.keys = keys
		}
	}
	return nil
}

//Compare and delete users that may nolonger be there
// Set the IAM users
func (u userMap) setIamUsers(g string) error {
	r := iam.IAMsvc.FetchGroup(g)
	for _, user := range iam.IAMsvc.GetIamUsers(r) {
		u[user] = &userData{group: g}
	}
	return nil
}

// Take in the channels and loop continuously through until we need to break
// syncing uses with whatever the time interval supplied is
func (u userMap) process(grp string, doneChan chan bool, stopChan chan bool, interval int) {
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
func (u userMap) userSync(grp string) error {
	// reset userMap map so it's empty
	u = make(map[string]*userData)

	// Fetch all iam users from group and assign to userMap type
	if err := u.setIamUsers(grp); err != nil {
		return err
	}

	// Set all the keys for users
	if err := u.setKey(); err != nil {
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

	if *onetime {
		if err := umap.userSync(*group); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Set the channels
	stopChan := make(chan bool)
	doneChan := make(chan bool)
	errChan := make(chan error, 10)

	go umap.process(*group, doneChan, stopChan, *interval)
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
