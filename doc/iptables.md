# Typical iptables used by Istio


In CloudRun/docker, the input is not captured - the infrastructure only sends H2 tunnels to the default HBONE port.

Should be applied with "iptables-restore -n" ( to not remove other iptables settings)
```shell

*nat
#:PREROUTING ACCEPT 
#:INPUT ACCEPT 
#:OUTPUT ACCEPT 
#:POSTROUTING ACCEPT 

#:ISTIO_INBOUND 
:ISTIO_IN_REDIRECT 
:ISTIO_OUTPUT 
:ISTIO_REDIRECT 

# Leftovers, not used.
#-A ISTIO_INBOUND -p tcp -m tcp --dport 15008 -j RETURN


-A OUTPUT -p tcp -j ISTIO_OUTPUT

# What gets excluded
# This is not needed unless inbound is captured. Istio uses it to differentiate inbound traffic from envouy to app.
#-A ISTIO_OUTPUT -s 127.0.0.6/32 -o lo -j RETURN

-A ISTIO_OUTPUT ! -d 127.0.0.1/32 -o lo -m owner --uid-owner 1337 -j ISTIO_IN_REDIRECT
-A ISTIO_OUTPUT -o lo -m owner ! --uid-owner 1337 -j RETURN
# Not needed, gid-owner sufficient.
# -A ISTIO_OUTPUT -m owner --uid-owner 1337 -j RETURN

-A ISTIO_OUTPUT ! -d 127.0.0.1/32 -o lo -m owner --gid-owner 1337 -j ISTIO_IN_REDIRECT

# Not envoy, on loopback - not intercepted.
-A ISTIO_OUTPUT -o lo -m owner ! --gid-owner 1337 -j RETURN

# Output from envoy is not intercepted 
-A ISTIO_OUTPUT -m owner --gid-owner 1337 -j RETURN

# 
-A ISTIO_OUTPUT -d 127.0.0.1/32 -j RETURN

# What actually gets captured
-A ISTIO_OUTPUT -d 10.0.0.0/8 -j ISTIO_REDIRECT
-A ISTIO_OUTPUT -j RETURN

-A ISTIO_IN_REDIRECT -p tcp -j REDIRECT --to-ports 15006
-A ISTIO_REDIRECT -p tcp -j REDIRECT --to-ports 15001

COMMIT

```

Most of the options do not apply for cloud-run-mesh, except:

## Istio defaults

ENVOY_PORT=
INBOUND_CAPTURE_PORT=
ISTIO_INBOUND_INTERCEPTION_MODE=
ISTIO_INBOUND_TPROXY_ROUTE_TABLE=
ISTIO_INBOUND_PORTS=
ISTIO_OUTBOUND_PORTS=
ISTIO_LOCAL_EXCLUDE_PORTS=
ISTIO_EXCLUDE_INTERFACES=
ISTIO_SERVICE_CIDR=
ISTIO_SERVICE_EXCLUDE_CIDR=
ISTIO_META_DNS_CAPTURE=
PROXY_PORT=15001
PROXY_INBOUND_CAPTURE_PORT=15006
PROXY_TUNNEL_PORT=15008
PROXY_UID=128
PROXY_GID=128
INBOUND_INTERCEPTION_MODE=
INBOUND_TPROXY_MARK=1337
INBOUND_TPROXY_ROUTE_TABLE=133
INBOUND_PORTS_INCLUDE=
INBOUND_PORTS_EXCLUDE=
OUTBOUND_IP_RANGES_INCLUDE=
OUTBOUND_IP_RANGES_EXCLUDE=
OUTBOUND_PORTS_INCLUDE=
OUTBOUND_PORTS_EXCLUDE=
KUBE_VIRT_INTERFACES=
ENABLE_INBOUND_IPV6=false
DNS_CAPTURE=false
CAPTURE_ALL_DNS=false
DNS_SERVERS=[],[]
OUTPUT_PATH=
NETWORK_NAMESPACE=
CNI_MODE=false
EXCLUDE_INTERFACES=

## hbone defaults

Required "-u 1337"



# Inbound story

All incoming traffic is received on the H2C port - infrastructure blocks all other ports. As such, we don't need anything
in the inbound path for iptables.

Instead the PORTS setting is used to control how HBone will forward to local ports, default is http:8080 (with direct 
access from the internet possible if allow_unauthenticated is set).

# TD Iptables

Current script support the following options:
-p PORT -u UID -g GID [-m mode] [-b ports] [-d ports] [-i CIDR] [-x CIDR] [-k interfaces]

Defaults or not applicable:
- UID/GID and PORT will be using the same values as in Istio, 1337 and 15001.
- '-m' Only supported mode is redirect, TPROXY requires NET_ADMIN permissions on the workload.
- the '-b' inbound ports to redirect, * or empty list supported. Not used in cloud-run-mesh or with hbone.
- '-k' - may be useful for VMs, in docker there is a single interface.
- '-d' - exclude inbound ports when '-b*'. 
- '-i' - out redirect by CIDR. Not used, we capture everything
- '-x' - exclude from out redirect.

As such the entire script can be directly replaced with the iptables-restore and the base Istio config. 
