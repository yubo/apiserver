#!/usr/bin/env bash

CN_BASE="proxy_tests"

cat > server.conf << EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
subjectAltName = @alt_names
[alt_names]
IP.1 = 127.0.0.1
EOF

cat > client.conf << EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
subjectAltName = @alt_names
[alt_names]
IP.1 = 127.0.0.1
EOF

# Create a certificate authority
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -days 100000 -out ca.crt -subj "/CN=${CN_BASE}_ca"

# Create a server certiticate
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/CN=${CN_BASE}_server" -config server.conf
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 100000 -extensions v3_req -extfile server.conf

# Create a client certiticate
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr -subj "/CN=tom/O=dev/O=admin" -config client.conf
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 100000 -extensions v3_req -extfile client.conf

# Clean up after we're done.
#rm ./*.pem
rm ./*.csr
rm ./*.srl
rm ./*.conf
