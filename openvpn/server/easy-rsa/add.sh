#!/bin/bash

if [ $# -ne 2 ]; then
    echo "0"
    exit
fi

if [ -f "/etc/openvpn/server/easy-rsa/pki/private/$1.key" ]; then
    echo "0"
    exit
fi

if [ -f "/etc/openvpn/server/easy-rsa/pki/reqs/$1.req" ]; then
    echo "0"
    exit
fi

if [ -f "/etc/openvpn/server/easy-rsa/pki/issued/$1.crt" ]; then
    echo "0"
    exit
fi

cd /etc/openvpn/server/easy-rsa
echo -e "\n" | ./easyrsa gen-req $1 nopass > /dev/null
echo -e "yes" | ./easyrsa sign-req client $1  > /dev/null
echo "ifconfig-push $2 255.255.255.0" > "/etc/openvpn/ccd/$1"
echo "1"
exit

