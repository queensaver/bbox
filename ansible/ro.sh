#!/bin/bash

set -e
set -o pipefail

sudo apt update
sudo apt install -y ansible git

# todo: git clone

git clone https://github.com/wogri/bbox.git

ansible-playbook bbox/ansible/read-only.yml
