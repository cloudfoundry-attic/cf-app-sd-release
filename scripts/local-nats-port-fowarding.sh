#!/bin/bash


ssh -L 8222:10.0.16.19:8222 -o "GatewayPorts yes" -l jumpbox  -i /tmp/toque-ssh-key 104.196.54.192
ssh -L 4222:10.0.16.19:4222 -l jumpbox  -i /tmp/toque-ssh-key 104.196.54.192