#!/bin/bash

if [ $# -ne 1 ]; then
    echo "0"
    exit
fi

if [ ! -f "/etc/openvpn/server/easy-rsa/pki/private/$1.key" ]; then
    echo "0"
	exit
fi


if [ ! -f "/etc/openvpn/server/easy-rsa/pki/issued/$1.crt" ]; then
    echo "0"
	exit
fi

if [ ! -f "/etc/openvpn/server/easy-rsa/pki/ca.crt" ]; then
    echo "0"
	exit
fi

if [ ! -f "/etc/openvpn/server/easy-rsa/pki/ta.key" ]; then
    echo "0"
	exit
fi

if [ ! -f "/etc/openvpn/ccd/$1" ]; then
    echo "0"
	exit
fi

cat /etc/openvpn/client/openvpn.txt > /etc/openvpn/client/$1.ovpn

echo "<key>" >> /etc/openvpn/client/$1.ovpn
cat /etc/openvpn/server/easy-rsa/pki/private/$1.key >> /etc/openvpn/client/$1.ovpn
echo "</key>" >> /etc/openvpn/client/$1.ovpn

echo "<cert>" >> /etc/openvpn/client/$1.ovpn
cat /etc/openvpn/server/easy-rsa/pki/issued/$1.crt >> /etc/openvpn/client/$1.ovpn
echo "</cert>" >> /etc/openvpn/client/$1.ovpn

echo "<ca>" >> /etc/openvpn/client/$1.ovpn
cat /etc/openvpn/server/easy-rsa/pki/ca.crt >> /etc/openvpn/client/$1.ovpn
echo "</ca>" >> /etc/openvpn/client/$1.ovpn

echo "<tls-auth>" >> /etc/openvpn/client/$1.ovpn
cat /etc/openvpn/server/easy-rsa/pki/ta.key >> /etc/openvpn/client/$1.ovpn
echo "</tls-auth>" >> /etc/openvpn/client/$1.ovpn

exit

