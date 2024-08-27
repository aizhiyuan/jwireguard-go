#!/bin/bash

if [ $# -ne 2 ]; then
    echo "0"
    exit
fi

if [ -f "/etc/openvpn/ccd/$1" ]; then
    echo "ifconfig-push $2 255.255.255.0" > "/etc/openvpn/ccd/$1"
fi
echo "1"
exit
