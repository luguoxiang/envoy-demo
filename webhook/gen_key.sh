openssl genrsa -out cert.key 2048
openssl req -x509 -new -nodes -key cert.key -subj "/CN=envoy-demo.default.svc" -days 10000 -out cert.crt
cat cert.crt|base64
