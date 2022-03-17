
```shell
gcloud config set project ${PROJECT_ID}
gcloud services enable \
    multiclusteringress.googleapis.com \
    gkehub.googleapis.com \
    cloudresourcemanager.googleapis.com \
    trafficdirector.googleapis.com \
    dns.googleapis.com
    --project=${PROJECT_ID}
    
   gcloud container hub multi-cluster-services enable 
   gcloud container hub multi-cluster-services describe
   gcloud container hub memberships list
   
    gcloud --project costin-asm1 container hub memberships register big1 --gke-cluster us-central1-c/big1 --enable-workload-identity
    
    gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member "serviceAccount:${PROJECT_ID}.svc.id.goog[gke-mcs/gke-mcs-importer]" \
    --role "roles/compute.networkViewer"
```
