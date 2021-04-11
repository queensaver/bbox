#!/bin/bash

set -e
set -o pipefail

sudo apt update
sudo apt install -y ansible git

if [ -e bbox ]; then
  git -C bbox/ pull
else
  git clone https://github.com/btelemetry/bbox.git
fi

ansible-playbook bbox/ansible/bbox.yml
ansible-playbook bbox/ansible/read_only.yml
sudo reboot
