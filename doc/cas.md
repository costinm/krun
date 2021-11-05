# CAS - the google CA service

- https://privateca.googleapis.com/v1/{parent=projects/*/locations/*}/caPools:fetchCaCerts - get trust anchor
- /v1/{parent=projects/*/locations/*/caPools/*}/certificates - create a certificate
- templates available, IAM

```js
{
  "caCerts": [
    {
      "PEM encoded CA"
    }
  ]
}

{
    "lifetime": "24h",
    "certificateTemplate": string,
    "subjectMode": "REFLECTED_SPIFFE",
    
    "pemCertificate": string,
    "certificateDescription": {
    object (CertificateDescription)
    },
    "pemCertificateChain": [
    string
    ],
    "createTime": string,
    "updateTime": string,
    "labels": {
    string: string,
    ...
    },
    
    // Union field certificate_config can be only one of the following:
    "pemCsr": string,
    "config": {
    object (CertificateConfig)
    }
}
```



Permissions: 
  privateca.certificates.create,
  roles/privateca.certificateRequester - arbitrary certificate

 privateca.certificates.createForSelf
  roles/privateca.workloadCertificateRequester

  roles/privateca.templateUser - get and use template. 


If using KMS certs - service-PROJECT_NUMBER@gcp-sa-privateca.iam.gserviceaccount.com must be granted permission 
to the key, roles/cloudkms.signerVerifier and roles/viewer


If subjectMode is DEFAULT = the Subject and/or SANs are specified in request, 'create' permission required.


```shell

gcloud beta services identity create --service=privateca.googleapis.com --project=PROJECT_ID
gcloud kms keys add-iam-policy-binding 'CRYPTOKEY_NAME' \
  --keyring='KEYRING_NAME' \
  --location='LOCATION' \
  --member='serviceAccount:service-PROJECT_NUMBER@gcp-sa-privateca.iam.gserviceaccount.com' \
  --role='roles/cloudkms.signerVerifier'
  
gcloud kms keys add-iam-policy-binding 'CRYPTOKEY_NAME' \
  --keyring='KEYRING_NAME' \
  --location='LOCATION' \
  --member='serviceAccount:service-PROJECT_NUMBER@gcp-sa-privateca.iam.gserviceaccount.com' \
  --role='roles/viewer'
  
gsutil iam ch serviceAccount:service-PROJECT_NUMBER@gcp-sa-privateca.iam.gserviceaccount.com:roles/storage.objectAdmin gs://BUCKET_NAME
```

## Profiles:

root_unconstrained -  can create CA sub-cert
subordinate_mtls_pathlen_0

## gRPC

Requires metadata: `x-goog-request-params: 'parent: projects/PROJECT_ID/locations/LOCATION_ID'`

## Quota:

For 'devOps' - 25 certs per second. '


Key-algo: ec-p256-sha256, rsa-pkcs1-4096-sha256
