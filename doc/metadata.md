# GCP metadata 

If running on GCP VM or CloudRun we can autoconfigure and get ID tokens using the metadata server.

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

In cloudrun additional env variables are made available and can be used for autoconfiguration.

- K_REVISION=fortio-00049-liw
- PORT
- K_SERVICE=fortio
- S2A_ACCESS_TOKEN ?
- K_CONFIGURATION=fortio

PORT defaults to 8080 - the applications are expected to use the PORT as listen address for their HTTP1/2. When krun
forks the app, it will override the PORT and set it to 8080. ( TODO: allow customization ).
When starting the app, PORT must be set to 15009, which is the 'hbone' h2c tunnel port.
