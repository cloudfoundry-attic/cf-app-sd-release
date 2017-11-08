#!/bin/bash


ssh -L 8822:10.0.16.19:8222 -o "GatewayPorts yes" -l jumpbox  -i /tmp/toque-ssh-key 104.196.54.192
ssh -L 8222:10.0.16.19:8222 -l bosh_09ee8a23605b4d5 -i /tmp/key 10.0.16.19


ssh -L 5222:10.0.16.19:4222 -l jumpbox  -i /tmp/toque-ssh-key 104.196.54.192
ssh -L 4222:10.0.16.19:4222 -l bosh_09ee8a23605b4d5 -i /tmp/key 10.0.16.19



# localhost ports: 8822 && 5222