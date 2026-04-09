CRT_LOCATION = certs
O = Mainflux
OU_CA = mainflux_ca
OU_CRT = mainflux_crt
EA = info@mainflux.com
CN_CA = Mainflux-CA
CN_SRV = localhost
CRT_FILE_NAME = thing

.PHONY: all ca server_cert thing_cert clean_certs

all: clean_certs ca server_cert

ca:
	openssl genrsa -out $(CRT_LOCATION)/ca.key 4096
	openssl req -new -x509 -days 3650 -key $(CRT_LOCATION)/ca.key \
		-out $(CRT_LOCATION)/ca.crt \
		-subj "/CN=$(CN_CA)/O=$(O)/OU=$(OU_CA)/emailAddress=$(EA)" \
		-addext "basicConstraints=critical,CA:true" \
		-addext "keyUsage=critical,digitalSignature,cRLSign,keyCertSign" \
		-addext "subjectKeyIdentifier=hash" \
		-addext "authorityKeyIdentifier=keyid:always,issuer"
	@echo "CA certificate generated with crlSign capability."
	@openssl x509 -in $(CRT_LOCATION)/ca.crt -noout -text | grep -A 3 "Key Usage"

server_cert:
	openssl req -new -sha256 -newkey rsa:4096 -nodes \
		-keyout $(CRT_LOCATION)/mainfluxlabs-server.key \
		-out $(CRT_LOCATION)/mainfluxlabs-server.csr \
		-subj "/CN=$(CN_SRV)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"
	openssl x509 -req -days 1000 \
		-in $(CRT_LOCATION)/mainfluxlabs-server.csr \
		-CA $(CRT_LOCATION)/ca.crt -CAkey $(CRT_LOCATION)/ca.key \
		-CAcreateserial -out $(CRT_LOCATION)/mainfluxlabs-server.crt
	rm $(CRT_LOCATION)/mainfluxlabs-server.csr

clean_certs:
	rm -f $(CRT_LOCATION)/*.crt $(CRT_LOCATION)/*.key $(CRT_LOCATION)/*.srl
