# Using a Secret for initial config

CloudRun allows mounting secrets, using

```shell
gcloud beta run deploy SERVICE --image IMAGE_URL  \
--update-secrets=PATH=project/PROJECT_NUMBER/secrets/SECRET_NAME:VERSION
```

Secret is max 64k - can be a yaml file.

For mesh, a similar naming would be
"--mesh=NAMESPACE:project/CONFIG_PROJECTNAME[/location/CONFIG_LOCATION/name/CONFIG_CLUSTER_NAME]"

or KSA.NAMESPACE@NAME.PROJECT
