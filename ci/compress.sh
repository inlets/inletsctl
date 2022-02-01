#!/bin/sh

cd bin
for f in inletsctl*; do tar -cvzf ../uploads/$f.tgz $f; done
