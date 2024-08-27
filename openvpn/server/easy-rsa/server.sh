#!/bin/bash

cd /etc/openvpn/server/easy-rsa
echo -e "yes" | ./easyrsa init-pki
echo -e "\n" | ./easyrsa build-ca nopass
echo -e "\n" | ./easyrsa gen-req server nopass
echo -e "yes" | ./easyrsa sign server server
./easyrsa gen-dh
openvpn --genkey --secret ./pki/ta.key 
rm -rf /etc/openvpn/ccd/*
rm -rf /etc/openvpn/client/*.ovpn
echo "1"
exit

