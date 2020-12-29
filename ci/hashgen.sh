#!/bin/sh
cd bin
for f in inletsctl*; do shasum -a 256 $f > ../uploads/$f.sha256; done
