package sync_iam

import (
	"fmt"
	"strings"

	"github.com/uswitch/aws_usersync/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

var (
	users []string
)

func GetIamClient(cfg *aws.Config) *iam.IAM {
	sess := session.Must(session.NewSession())
	isvc := iam.New(sess, cfg)
	return isvc
}

func getUserKey(user string, svc *iam.IAM, k []*iam.SSHPublicKeyMetadata) ([]string, error) {
	encoding := "SSH"
	publicKeys := []string{}
	for _, key := range k {
		kresp, err := svc.GetSSHPublicKey(&iam.GetSSHPublicKeyInput{
			Encoding:       aws.String(encoding),
			SSHPublicKeyId: key.SSHPublicKeyId,
			UserName:       aws.String(user),
		})
		log.Debug(fmt.Sprintf("Called AWS GetSSHPublicKey: user %v for %v using encoding %v", user, key.SSHPublicKeyId, encoding))
		if err != nil {
			return publicKeys, err
		}
		if *kresp.SSHPublicKey.Status == "Active" {
			log.Debug(fmt.Sprintf("Got active key from AWS %v", *kresp.SSHPublicKey.SSHPublicKeyBody))
			publicKeys = append(publicKeys, *kresp.SSHPublicKey.SSHPublicKeyBody)
		}
	}
	return publicKeys, nil
}

func GetKeys(user string, svc *iam.IAM) ([]string, error) {
	resp, err := svc.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{
		UserName: aws.String(user),
	})
	if err != nil {
		log.Error(fmt.Sprintf("Error getting keys for user %v", user))
		return nil, err
	}
	if len(resp.SSHPublicKeys) > 0 {
		ukey, err := getUserKey(user, svc, resp.SSHPublicKeys)
		if err != nil {
			log.Error("Error calling getUserKey")
			return nil, err
		} else {
			return ukey, nil
		}
	}
	return nil, nil
}

func GetIamUsers(r *iam.GetGroupOutput) []string {
	for _, user := range r.Users {
		users = append(users, strings.ToLower(*user.UserName))
	}
	log.Debug(fmt.Sprintf("Got a list of Iam Users %v", users))
	return users
}
