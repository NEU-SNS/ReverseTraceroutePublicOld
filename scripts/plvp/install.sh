#!/bin/sh

cp ./init/plvp /etc/rc.d/init.d/
./post-install.sh

/sbin/service plvp start

