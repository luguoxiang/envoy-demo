#!/bin/sh

# Create a new chain for redirecting outbound traffic to the Envoy port.
# In both chains, '-j RETURN' bypasses Envoy and '-j ENVOY_REDIRECT'
# redirects to Envoy.
iptables -t nat -N ENVOY_REDIRECT
iptables -t nat -A ENVOY_REDIRECT -p tcp -j REDIRECT --to-port ${PROXY_PORT}

# Use this chain also for redirecting inbound traffic to the common Envoy port
# when not using TPROXY.
iptables -t nat -N ENVOY_IN_REDIRECT
iptables -t nat -A ENVOY_IN_REDIRECT -p tcp -j REDIRECT --to-port ${PROXY_PORT}

# Create a new chain for selectively redirecting inbound packets to Envoy.
iptables -t nat -N ENVOY_INBOUND

# Jump to the ENVOY_INBOUND chain from PREROUTING chain for all tcp traffic.
iptables -t nat -A PREROUTING -p tcp -j ENVOY_INBOUND

for port in ${INBOUND_PORTS_INCLUDE}; do
  iptables -t nat -A ENVOY_INBOUND -p tcp --dport ${port} -j ENVOY_IN_REDIRECT
done

# Create a new chain for selectively redirecting outbound packets to Envoy.
iptables -t nat -N ENVOY_OUTPUT

# Jump to the ENVOY_OUTPUT chain from OUTPUT chain for all tcp traffic.
iptables -t nat -A OUTPUT -p tcp -j ENVOY_OUTPUT

# Avoid infinite loops. Don't redirect Envoy traffic directly back to
# Envoy for non-loopback traffic.
iptables -t nat -A ENVOY_OUTPUT -p tcp --dport ${PROXY_MANAGE_PORT} -j RETURN
iptables -t nat -A ENVOY_OUTPUT -m owner --uid-owner ${PROXY_UID} -j RETURN
iptables -t nat -A ENVOY_OUTPUT -m owner --gid-owner ${PROXY_UID} -j RETURN

# Redirect remaining outbound traffic to Envoy
iptables -t nat -A ENVOY_OUTPUT -j ENVOY_REDIRECT

# Skip redirection for Envoy-aware applications and
# container-to-container traffic both of which explicitly use
# localhost.
iptables -t nat -A ENVOY_OUTPUT -d 127.0.0.1/32 -j RETURN
iptables -L -t nat 
