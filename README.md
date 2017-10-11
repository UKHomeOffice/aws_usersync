# aws_usersync

[![Build Status](https://travis-ci.org/uswitch/aws_usersync.svg?branch=master)](https://travis-ci.org/uswitch/aws_usersync)

This is used for syncing users from AWS to the local machine as well as their user key. It runs as a daemon and polls with whatever interval you define. By default it is set to run only once and exit, but this can be overriden.

The default group people are added to is sudo, but this can be overriden to add users to an alternative group.

This was written primarily to only work with AWS and also CoreOS. The user keys are really for AWS CodeCommit service, however, as they are
associated with the IAM account, they become accessible through IAM. It isn't particularly obvious that you need to place your key there but this is where it needs to go. 

### AWS IAM

If a user logs in to AWS Console and goes to AWS IAM Identity Access Management and then their own user, there is the codecommit section at the bottom. Users can paste in their public key in there, or multiple and make them active. This tool will only sync active keys, if you make a key inactive, then it will replace the keys on the box with only the active ones. 

#### IAM POLICY

Below is the policy that needs to be associated with the instances you are provisioning. Once you have created this, you can assign this to instances to give the relevant access to the instance to get the keys / users / groups.

```
{
   "Version": "2012-10-17",
   "Statement": [
       {
           "Sid": "Stmt1442396947000",
           "Effect": "Allow",
           "Action": [
               "iam:GetGroup",
               "iam:GetSSHPublicKey",
               "iam:GetUser",
               "iam:ListSSHPublicKeys"
           ],
           "Resource": [
               "arn:aws:iam::*"
           ]
       }
   ]
}
```


### How to use this

You can build the go application by running:
```
git clone git@github.com:uswitch/aws_usersync.git
cd aws_usersync
docker run --rm -it -v "$PWD":/go -w /go quay.io/uswitchdigital/go-gb:1.0.0 gb build all
```

This will build the application in the current directory creating a bin/aws_usersync binary

To run the application you need to set environment variables or have relevant access to IAM:

```
export AWS_ACCESS_KEY_ID=12345678893
export AWS_SECRET_ACCESS_KEY=30302499439434
./bin/aws_usersync -g="group1,group2,group3"
```

This will grab all the users within that group and add them locally with relevant keys as a one time run, to run this at an interval of 5 minutes

```
export AWS_ACCESS_KEY_ID=12345678893
export AWS_SECRET_ACCESS_KEY=30302499439434
./bin/aws_usersync -o=false -i=5 -g="group1,group2,group3"
```

For debugging issues you can run it in debug mode
```
./bin/aws_usersync -o=false -i=5 -g="group1,group2,group3" -L=debug
```

##### Notes
This is new and needs some cleanup on the code really and improving as it's a bit jumbled together in areas, but there are retrospective
issues raised for things, to clean things up. 


