#!/bin/bash 

# A utiliser avant de commencer le programme répartit 
nettoyer() {
	echo "Nettoyage..."
	killall application 2>/dev/null
	killall control 2>/dev/null
	killall tee 2>/dev/null
	killall cat 2>/dev/null
	rm -f /tmp/in_app* /tmp/out_app* /tmp/in_ctl* /tmp/out_ctl*
	exit 0
}
trap nettoyer INT QUIT TERM