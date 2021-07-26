# Opinionated zero-config (WIP)

The opinionated config is based on detecting the environment and relying on
defined locations and naming.

The intended zero config will use:

- if KUBECONFIG env variable is checked first and used if found.
- metadata server will be used to get 'project-id', 'region' 
- metadata server 'id' will be used as pod id suffix

For namespace and service:
- K_SERVICE

