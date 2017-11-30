package sync_iam

import (
	"fmt"
	"strings"

	"github.com/UKHomeOffice/aws_usersync/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var IAMsvc *IAM

type IAM struct {
	iamiface.IAMAPI
}

func NewIAM(awsSession *session.Session, cfg *aws.Config) {
	IAMsvc = &IAM{iam.New(awsSession, cfg)}
}

func (i *IAM) FetchGroup(g string) *iam.GetGroupOutput {
	rp, err := i.GetGroup(&iam.GetGroupInput{GroupName: aws.String(g)})
	if err != nil {
		log.Error(fmt.Sprintf("Error getting Group: %v, %v", g, err))
	}
	return rp
}

func (i *IAM) getUserKey(user string, k []*iam.SSHPublicKeyMetadata) ([]string, error) {
	encoding := "SSH"
	publicKeys := []string{}
	for _, key := range k {
		kresp, err := i.GetSSHPublicKey(&iam.GetSSHPublicKeyInput{
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

func (i *IAM) GetKeys(user string) ([]string, error) {
	resp, err := i.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{
		UserName: aws.String(user),
	})
	if err != nil {
		log.Error(fmt.Sprintf("Error getting keys for user %v", user))
		return nil, err
	}
	if len(resp.SSHPublicKeys) > 0 {
		ukey, err := i.getUserKey(user, resp.SSHPublicKeys)
		if err != nil {
			log.Error("Error calling getUserKey")
			return nil, err
		} else {
			return ukey, nil
		}
	}
	return nil, nil
}

func (i *IAM) GetIamUsers(r *iam.GetGroupOutput) []string {
	var users []string
	for _, user := range r.Users {
		users = append(users, strings.ToLower(*user.UserName))
	}
	log.Debug(fmt.Sprintf("Got a list of Iam Users %v", users))
	return users
}
