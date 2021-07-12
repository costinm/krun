#!/bin/bash

export HTTP_PROXY=127.0.0.1:15080
export POD_NAMESPACE=fortio
export POD_NAME=fortio-cr
export LABEL_APP=fortio-cr

export PROJECT=wlhe-cr
export LOCATION=us-central1
export CLUSTER=istio
export XDS_ADDR=istiod.wlhe.i.webinf.info:443

krun /bin/bash
