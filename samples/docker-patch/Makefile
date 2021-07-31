# Local settings
ISTIOD?=$(shell cd ../..; pwd)
-include ${ISTIOD}/.local.mk

# GKE cluster used
PROJECT_ID=wlhe-cr
LOCATION=us-central1-c
CLUSTER=istio

# Region where CR will be deployed
REGION=us-central1

# Where to store the images
REPO=gcr.io/wlhe-cr

# The install assumes a subdomain is configured as a
# *.SUBDOMAIN A IP_ADDRESS
# where IP_ADDRES is the Istiod Ingress gateway address.
#
# In addition, CertManager/etc ACME certificates and routes are set for
# istiod.SUBDOMAIN
SUBDOMAIN=wlhe.i.webinf.info

#SVC=proxybase-dmgksid7iq-uc.a.cloud-run-qual.sandbox.google.com

SVC=fortio-dmgksid7iq-uc.a.cloud-run-qual.sandbox.google.com
######

# Suffix for the CR app
CR_SUFFIX=-icq63pqnqq-uc.a.run.app

ISTIOD_URL?=istiod.${SUBDOMAIN}:443

NS=fortio

all: push-fortio

istio-fortio: patch-fortio

push-fortio: docker-fortio
	docker push ${REPO}/fortio:latest

# Local build - using ko.local for krun, local docker build
local/all: push-fortio-local deploy-fortio

push-fortio-local:
	(cd ../..; make local/krun)
	docker build . -f Dockerfile.fortio \
		--build-arg BASE=ko.local/krun:latest \
		-t ${REPO}/fortio:latest
	docker push ${REPO}/fortio:latest

# Build the image using the proxy as base
docker-fortio:
	docker pull ghcr.io/costinm/krun/krun:latest
	docker build . -f Dockerfile.fortio -t ${REPO}/fortio:latest

# WIP - use the app's base image and add the proxy
patch-fortio:
	docker pull ghcr.io/costinm/krun/krun:latest
	docker build . -f ../Dockerfile.istio-patch \
       --build-arg NS=fortio \
       --build-arg APP=cr-fortio \
       --build-arg BASE=ubuntu \
       -t ${REPO}/istio-fortio:latest

#       --build-arg BASE=costinm/fortio:latest \

push-istio-fortio: patch-fortio
	docker push ${REPO}/istio-fortio:latest




