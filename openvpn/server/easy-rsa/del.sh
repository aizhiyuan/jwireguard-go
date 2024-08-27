#!/bin/bash

if [ $# -ne 1 ]; then
    echo "0"
    exit
fi

if [ -f "/etc/openvpn/server/easy-rsa/pki/private/$1.key" ]; then
    rm -rf /etc/openvpn/server/easy-rsa/pki/private/$1.key
fi

if [ -f "/etc/openvpn/server/easy-rsa/pki/reqs/$1.req" ]; then
    rm -rf /etc/openvpn/server/easy-rsa/pki/reqs/$1.req
fi

if [ -f "/etc/openvpn/server/easy-rsa/pki/issued/$1.crt" ]; then
    rm -rf /etc/openvpn/server/easy-rsa/pki/issued/$1.crt
fi

if [ -f "/etc/openvpn/ccd/$1" ]; then
    rm -rf /etc/openvpn/ccd/$1
fi

if [ -f "/etc/openvpn/client/$1.ovpn" ]; then
    rm -rf /etc/openvpn/client/$1.ovpn
fi

echo "1"
exit

