iptables  -t nat -D OUTPUT  -p tcp -j ENVOY_OUTPUT
iptables  -t nat --flush ENVOY_OUTPUT
iptables  -t nat -X ENVOY_OUTPUT
iptables  -t nat --flush ENVOY_REDIRECT
iptables  -t nat -X ENVOY_REDIRECT

iptables -t nat -D PREROUTING -p tcp -j ENVOY_INBOUND
iptables  -t nat --flush ENVOY_INBOUND
iptables  -t nat -X ENVOY_INBOUND
iptables  -t nat --flush ENVOY_IN_REDIRECT
iptables  -t nat -X ENVOY_IN_REDIRECT

iptables  -t nat -L
