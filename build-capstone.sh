#!/bin/bash

CS_VERSION=4.0.1

mkdir -p third_party && cd third_party

wget https://github.com/aquynh/capstone/archive/${CS_VERSION}.tar.gz
tar xzf ./${CS_VERSION}.tar.gz
mv ./capstone-${CS_VERSION} ./capstone
cd ./capstone
./make.sh
