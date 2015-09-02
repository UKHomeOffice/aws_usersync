# aws_usersync

This is used for syncing users from AWS to the local machine as well as their user key. It runs as a daemon and polls with whatever interval you define. By default it is set to run only once and exit, but this can be overriden. 

This was written primarily to only work with AWS and also CoreOS. The user keys are really for AWS CodeCommit service, however, as they are
associated with the IAM account, they become accessible through IAM. It isn't particularly obvious that you need to place your key there but this is where it needs to go. 

# AWS IAM

If a user logs in to AWS Console and goes to AWS IAM Identity Access Management and then their own user, there is the codecommit section at the bottom. Users can paste in their public key in there, or multiple and make them active. This tool will only sync active keys, if you make a key inactive, then it will replace the keys on the box with only the active ones. 


*Note*
This is new and needs some cleanup on the code really and improving as it's a bit jumbled together in areas, but there are retrospective
issues raised for things, to clean things up. 


