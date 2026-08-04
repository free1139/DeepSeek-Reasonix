#!/bin/sh
set -e

make
cd bin
tar -czf reasonix.tar.gz reasonix
sudo cp reasonix.tar.gz /var/www/html
cd -
