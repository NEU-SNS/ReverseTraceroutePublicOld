/sbin/chkconfig --add plvp
[ -e /var/log/plvp ] && [ -f /var/log/plvp ] && rm /var/log/plvp
mkdir -p /var/log/plvp
