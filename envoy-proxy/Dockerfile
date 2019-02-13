FROM envoyproxy/envoy:latest
RUN apt-get update
RUN apt-get install apt-utils -y
RUN apt-get install iptables -y
RUN apt-get install python -y
RUN apt-get install curl -y
RUN apt-get install sudo -y
RUN apt-get install net-tools -y
COPY dynamic-envoy.yaml /etc/envoy/envoy.yaml
COPY run.py .
COPY iptable_init.sh .
COPY iptable_clean.sh .
CMD ["python", "run.py", "/etc/envoy/envoy.yaml"]

