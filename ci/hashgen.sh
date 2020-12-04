#!/bin/sh
cd bin
for f in inletsctl*; do shasum -a 256 $f > $f.sha256; done
