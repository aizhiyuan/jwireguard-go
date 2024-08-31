#!/bin/bash

if [ $# -ne 1 ]; then
    echo "0"
    exit
fi

if [ -f "./pki/private/$1.key" ]; then
    rm -rf ./pki/private/$1.key
fi

if [ -f "./pki/reqs/$1.req" ]; then
    rm -rf ./pki/reqs/$1.req
fi

if [ -f "./pki/issued/$1.crt" ]; then
    rm -rf ./pki/issued/$1.crt
fi

if [ -f "../../ccd/$1" ]; then
    rm -rf ../../ccd/$1
fi

if [ -f "../../client/$1.ovpn" ]; then
    rm -rf ../../client/$1.ovpn
fi

echo "1"
exit

