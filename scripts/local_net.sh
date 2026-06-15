#!/usr/bin/env bash
#
# Script pour faire tourner notre système réparti en local
#
# Topologie en anneau de N=NB_SITES
# Chaque site embarque un binaire Go `net` pour gérer la couche réseau, un binaire Go `control`
# pour gérer les mécanismes de système réparti du cours (snapshot, horloges, etc.) et
# une `application`` (avec serveur web intégré depuis la fusion app+server)
#
# Pipeline pour chaque site i :
#   browser        <---WS--->   application_i (embarque le serveur HTTP/WS)
#   application_i  --stdout-->  control_i  (FIFO out_app_i -> in_ctl_i)
#   control_i      --stdout-->  application_i (local) + net_i
#   net_i          --stdout-->  control_i (local) + net_{i+1} (anneau)

NB_SITES=4
BASE_PORT=${1:-4444}

# Se placer à la racine du projet quel que soit le cwd d'appel
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

for bin in application control net; do
	if [[ ! -x "./bin/$bin" ]]; then
		echo "Binaire manquant : ./bin/$bin"
    	echo "Compilez avec 'make build'"
    	exit 1
	fi
done

nettoyer() {
	echo "Nettoyage..."
	killall application 2>/dev/null
	killall control 2>/dev/null
	killall net 2>/dev/null
	killall tee 2>/dev/null
	killall cat 2>/dev/null
	rm -f /tmp/in_app* /tmp/out_app* /tmp/in_ctl* /tmp/out_ctl* /tmp/in_net* /tmp/out_net*
	exit 0
}
trap nettoyer INT QUIT TERM

# On setup les FIFOs de notre site Init
rm -f /tmp/in_app1 /tmp/out_app1 /tmp/in_ctl1 /tmp/out_ctl1 /tmp/in_net1 /tmp/out_net1
mkfifo /tmp/in_app1 /tmp/out_app1
mkfifo /tmp/in_ctl1 /tmp/out_ctl1
mkfifo /tmp/in_net1 /tmp/out_net1

# On lance les processus de notre site Init
./bin/application -n "app1" -id "J1" -addr localhost -port "$BASE_PORT" -web "$ROOT/web" < /tmp/in_app1 > /tmp/out_app1 &
./bin/control -n "ctl1" -id "1" -sites "1" -isInitiator < /tmp/in_ctl1 > /tmp/out_ctl1 &

# On commence à créer les redirections app/ctl de notre site Init
cat /tmp/out_app1 > /tmp/in_ctl1 &
tee /tmp/in_app1 < /tmp/out_ctl1 > /tmp/in_net1 &
cat /tmp/out_net1 > /dev/null &
TTOUT_1=$!
./bin/net -n "net1" -id "1" -next "1" -ttout "$TTOUT_1" -static < /tmp/in_net1 > /tmp/out_net1 &

echo " * http://localhost:$BASE_PORT -> J1"

for i in $(seq 2 $NB_SITES); do
	sleep 1
	./scripts/join_site.sh "$i" 1 "$BASE_PORT"
done

sleep 3600
nettoyer