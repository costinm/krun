# Useful GCP metadata 

curl -H "Metadata-Flavor: Google" \
  'http://metadata/computeMetadata/v1/instance/service-accounts/default/identity?audience=https://www.example.com'


http://metadata.google.internal/computeMetadata/v1/project/

- project-id
- numeric-project-id

http://metadata.google.internal/computeMetadata/v1/instance/

- id - long UUID
- region - projects/NUM/regions/us-central1
- zone - projects/NUM/zones/us-central1-1
- service-accounts
    - email - 601426346923-compute@developer.gserviceaccount.com
    - ?audience=... - ID token - includes the same email, "iss": "https://accounts.google.com", sub and azp
    - token - access token for GCP
    
# CloudRun env

- K_REVISION=fortio-00049-liw
- PORT
- K_SERVICE=fortio
- S2A_ACCESS_TOKEN ?
- K_CONFIGURATION=fortio
