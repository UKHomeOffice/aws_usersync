package coreos_iam

import (
  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/service/iam"
)

var (
  users []string
)

func GetIamClient(cfg *aws.Config) *iam.IAM {
  isvc := iam.New(cfg)
  return isvc
}

func getUserKey(user string, svc *iam.IAM, k []*iam.SSHPublicKeyMetadata) ([]string, error) {
  encoding := "SSH"
  publicKeys := []string{}
  for _, key := range k {
    kresp, err := svc.GetSSHPublicKey(&iam.GetSSHPublicKeyInput{
      Encoding: aws.String(encoding),
      SSHPublicKeyId: key.SSHPublicKeyId,
      UserName: aws.String(user),
    })
    if err != nil {
      return publicKeys, err
    }
    if *kresp.SSHPublicKey.Status == "Active" {
      publicKeys = append(publicKeys, *kresp.SSHPublicKey.SSHPublicKeyBody)
    }
  }
  return publicKeys, nil
}

func GetKeys(user string, svc *iam.IAM) ([]string, error) {
  resp, err := svc.ListSSHPublicKeys(&iam.ListSSHPublicKeysInput{
    UserName: aws.String(user),
  });
  if err != nil {
    return nil, err
  }
  if len(resp.SSHPublicKeys) > 0 {
    ukey, err := getUserKey(user, svc, resp.SSHPublicKeys)
    if err != nil {
      return nil, err
    } else {
      return ukey, nil
    }
  }
  return nil, nil
}

func GetIamUsers(r *iam.GetGroupOutput) []string {
  for _, user  := range r.Users {
    users = append(users, *user.UserName)
  }
  return users
}
