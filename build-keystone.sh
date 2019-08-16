#!/bin/bash

KS_VERSION=0.9.1

mkdir -p third_party && cd third_party

wget https://github.com/keystone-engine/keystone/archive/${KS_VERSION}.tar.gz
tar xzf ./${KS_VERSION}.tar.gz
mv ./keystone-${KS_VERSION} ./keystone

cd ./keystone

if [[ "$OSTYPE" == "darwin"* ]]; then
    # MacOS does not support i386 build anymore.
    sed -i -e 's/i386;//' ./make-common.sh
fi

mkdir build
cd build
../make-lib.sh
