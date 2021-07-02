Small command line tool to init a Kubeconfig file for GCP VM, Cloudrun, or any other
environment with a metadata server producing GCP SA tokens.

The SA must have the IAM permissions to connect to the GKE cluster. This also works for 
GKE Connect Gateway, for non-GKE clusters registered in the hub.

Will generate a kube config file with all clusters, and Secrets for remote access.

Equivalent config using shell:

```shell
CMD="gcloud container clusters describe ${CLUSTER} --zone=${ZONE} --project=${PROJECT}"

K8SURL=$($CMD --format='value(endpoint)')
K8SCA=$($CMD --format='value(masterAuth.clusterCaCertificate)' )
```

```yaml
apiVersion: v1
kind: Config
current-context: my-cluster
contexts: [{name: my-cluster, context: {cluster: cluster-1, user: user-1}}]
users: [{name: user-1, user: {auth-provider: {name: gcp}}}]
clusters:
- name: cluster-1
  cluster:
    server: "https://${K8SURL}"
    certificate-authority-data: "${K8SCA}"

```
