#!/bin/bash

if [ $# -ne 2 ]; then
    echo "0"
    exit
fi

if [ -f "./pki/private/$1.key" ]; then
    echo "0"
    exit
fi

if [ -f "./pki/reqs/$1.req" ]; then
    echo "0"
    exit
fi

if [ -f "./pki/issued/$1.crt" ]; then
    echo "0"
    exit
fi

echo -e "\n" | ./easyrsa gen-req $1 nopass > /dev/null
echo -e "yes" | ./easyrsa sign-req client $1  > /dev/null
echo "ifconfig-push $2 255.255.0.0" > "../../ccd/$1"
echo "1"
exit

