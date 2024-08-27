#!/bin/bash

if [ $# -ne 2 ]; then
    echo "0"
fi

status=0

if [ -f "/etc/openvpn/server/easy-rsa/pki/private/$1.key" ]; then
    status=1
fi

if [ -f "/etc/openvpn/server/easy-rsa/pki/reqs/$1.req" ]; then
    status=1
fi

if [ -f "/etc/openvpn/server/easy-rsa/pki/issued/$1.crt" ]; then
    status=1
fi

cd /etc/openvpn/server/easy-rsa
if [ $status -ne 1];then
    echo -e "yes\n" | ./easyrsa gen-req $1 nopass > /dev/null
else
    echo -e "\n" | ./easyrsa gen-req $1 nopass > /dev/null
fi
echo -e "yes" | ./easyrsa sign-req client $1  > /dev/null
echo "ifconfig-push $2 255.255.255.0" > "/etc/openvpn/ccd/$1"
echo "1"
exit

