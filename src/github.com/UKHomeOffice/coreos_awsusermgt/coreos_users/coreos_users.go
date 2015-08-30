package coreos_users

import (
	"bufio"
	"os/user"
	"path/filepath"
	"os"
	"os/exec"
	"fmt"
	"strings"
	"strconv"
)

var Output []string

const (
	AuthorizedKeysFile = "authorized_keys"
	SSHDir             = ".ssh"
)

type awsUser struct {
	iamUser   string
	Group     string
	SudoGroup string
	Keys      []string
  localUser *user.User
}

// Initiate the user function
func New(user string, group string, sgroup string, keys []string) *awsUser {
	ustruct := &awsUser{
		iamUser:   user,
		Group:     group,
		SudoGroup: sgroup,
		Keys:      keys,
	}
	return ustruct
}

// sshDirPath returns the path to the .ssh dir for the user.
func sshDirPath(u *user.User) string {
	return filepath.Join(u.HomeDir, SSHDir)
}

// authKeysFilePath returns the path to the authorized_keys file for the user.
func authKeysFilePath(u *user.User) string {
	return filepath.Join(sshDirPath(u), AuthorizedKeysFile)
}

// Remove users from system that are not in the group list
func RemoveUser(usr string) error {
	u, err := user.Lookup(usr)
	if err != nil {
		return err
	}
	CMD := "userdel"
	CMD_ARGS := []string{"-r",  u.Username}
	if _, err := exec.Command(CMD, CMD_ARGS...).Output(); err != nil {
		return err
	}
	return nil
}

// Compare the keys to find what keys are missing locally compared to what is in IAM
func GetArrayDiff(k1 []string, k2 []string) ([]string) {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range k1 {
			found := false
			for _, s2 := range k2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			// Key not found so add it to difference
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			k1, k2 = k2, k1
		}
	}
	return diff
}

// Loop through the keys and call add key to add key to the box
func addKeys(l *user.User, kp string, ks []string) error {
	for _, key := range ks {
		Output = append(Output, fmt.Sprintf("Calling addkey to add key\n"))
		if err := addKey(l, kp, key); err != nil {
			return err
		}
	}
	return nil
}


// Add the users SSH key onto the system
func addKey(u *user.User, keypath string, key string) error {
	f, err := os.OpenFile(keypath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
	defer f.Close()
	if err != nil {
		return err
	}
	if _, err := f.WriteString(key); err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	if err := os.Chown(keypath, uid, gid); err != nil {
		return err
	}
	Output = append(Output, fmt.Sprintf("Adding key to %v for user %v\n", keypath, u.Username))
	return nil
}

// Add carriage return for new array elements
func addNewLine(s1 []string) []string {
	var newslice []string
	for i, e := range s1 {
		if i == 0 {
			newslice = append(newslice, e)
		} else {
			s := fmt.Sprintf("\n%v", e)
			newslice = append(newslice, s)
		}
	}
	return newslice
}
// Get the keys of user and add keys that are not already there including differences
func (l *awsUser) DoKeys() error {
	keyPath := authKeysFilePath(l.localUser)
 	keys, err := l.getKeys(keyPath)
	if err != nil {
		Output = append(Output, fmt.Sprintf("Users Key doesn't exist so adding\n"))
		keys := addNewLine(l.Keys)
		if err := addKeys(l.localUser, keyPath, keys); err != nil {
			return err
		}
	} else {
		Output = append(Output, fmt.Sprintf("User has Keys so finding differences\n"))
		diffkeys := GetArrayDiff(keys, l.Keys)
		keys = addNewLine(diffkeys)
		if err := addKeys(l.localUser, keyPath, keys); err != nil {
			return err
		}
	}
	return nil
}

// Check if there is the authorized keys file if it is then return all the keys from it
func (l *awsUser) getKeys(keyPath string) ([]string, error) {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil, err
	} else {
		kfile, err := os.Open(keyPath)
		if err != nil {
			return nil, err
		}
		defer kfile.Close()
		var keys []string
		scanner := bufio.NewScanner(kfile)
		for scanner.Scan() {
			keys = append(keys, scanner.Text())
		}
		return keys, scanner.Err()
	}
}

func GetAllUsers() ([]string, error) {
	passwd := "/etc/passwd"
	fpasswd, err := os.Open(passwd)
	if err != nil {
		return nil, err
	}
	defer fpasswd.Close()
	var users []string
	scanner := bufio.NewScanner(fpasswd)
	for scanner.Scan() {
		users = append(users, strings.Split(scanner.Text(), ":")[0])
	}
	return users, scanner.Err()
}

// Add user onto the system using useradd exec
func (l *awsUser) addUser() error {
	CMD_ARGS   := []string{"-p", "123", "-U", "-m", l.iamUser, "-G", l.SudoGroup}
	_, err := exec.Command("useradd", CMD_ARGS...).Output()
	if err != nil {
		return err
	} else {
		lusr, _ := user.Lookup(l.iamUser)
		l.localUser = lusr
	}
	out := fmt.Sprintf("Creating user %v", l.iamUser)
	Output = append(Output, out)
	return nil
}

// Sync all users and keys onto the coreos host this is the primary function
func (l *awsUser) Sync() ([]string, error) {
	if err := l.addUser(); err != nil {
		return nil, err
	}
	if err := l.DoKeys(); err != nil {
		return nil, err
	}
	return Output, nil
}
