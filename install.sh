#!/bin/bash
set -euo pipefail -x

# Create two repos
git clone https://github.com/ynoproject/ynoclient.git
cp -r ynoclient ynoclient_modified

# Frontend setup
git clone https://github.com/ynoproject/ynofront.git

# 2kki setup
cd ynofront
git clone https://github.com/ynoproject/forest-orb.git
mv forest-orb 2kki

# language setup
git clone https://github.com/ynoproject/ynotranslations.git
mv ynotranslations data

# Move all language dirs to "Language" subdir
# why do i have to do this?
ls -l  ./data | grep -v 'total' |  awk '{print $9}' | xargs -I'{}' /bin/bash -c 'mkdir -p ./data/{}/Language && mv ./data/{}/* ./data/{}/Language/' 