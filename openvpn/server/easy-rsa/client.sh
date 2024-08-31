#!/bin/bash

if [ $# -ne 1 ]; then
    echo "0"
    exit
fi

if [ ! -f "./pki/private/$1.key" ]; then
    echo "0"
	exit
fi


if [ ! -f "./pki/issued/$1.crt" ]; then
    echo "0"
	exit
fi

if [ ! -f "./pki/ca.crt" ]; then
    echo "0"
	exit
fi

if [ ! -f "./pki/ta.key" ]; then
    echo "0"
	exit
fi

if [ ! -f "../../ccd/$1" ]; then
    echo "0"
	exit
fi

cat ../../client/openvpn.txt > ../../client/$1.ovpn

echo "<key>" >> ../../client/$1.ovpn
cat ./pki/private/$1.key >> ../../client/$1.ovpn
echo "</key>" >> ../../client/$1.ovpn

echo "<cert>" >> ../../client/$1.ovpn
cat ./pki/issued/$1.crt >> ../../client/$1.ovpn
echo "</cert>" >> ../../client/$1.ovpn

echo "<ca>" >> ../../client/$1.ovpn
cat ./pki/ca.crt >> ../../client/$1.ovpn
echo "</ca>" >> ../../client/$1.ovpn

echo "<tls-auth>" >> ../../client/$1.ovpn
cat ./pki/ta.key >> ../../client/$1.ovpn
echo "</tls-auth>" >> ../../client/$1.ovpn

exit

