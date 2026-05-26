#!/usr/bin/env bash
#
# Script pour faire tourner notre système réparti en local
# Topologie : anneau de N=NB_SITES contrôles, chaque site embarque
# son application (avec serveur web intégré depuis la fusion app+server).
#
# Pipeline pour chaque site i :
#   browser <--WS--> application_i (embarque le serveur HTTP/WS)
#   application_i  --stdout-->  control_i  (FIFO out_app_i -> in_ctl_i)
#   control_i      --stdout-->  application_i (local)  +  control_{i+1} (anneau)

NB_SITES=3
BASE_PORT=${1:-4444}

# Se placer à la racine du projet quel que soit le cwd d'appel
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

for bin in application control; do
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
	killall tee 2>/dev/null
	killall cat 2>/dev/null
	rm -f /tmp/in_app* /tmp/out_app* /tmp/in_ctl* /tmp/out_ctl*
	exit 0
}
trap nettoyer INT QUIT TERM

# Pour chaque site, on créé 2 fifos (in et out) pour l'app et le contrôle
for i in $(seq 1 $NB_SITES); do
	rm -f /tmp/in_app$i /tmp/out_app$i /tmp/in_ctl$i /tmp/out_ctl$i
	mkfifo /tmp/in_app$i  /tmp/out_app$i
	mkfifo /tmp/in_ctl$i  /tmp/out_ctl$i
done

# Lancement des processus
for i in $(seq 1 $NB_SITES); do
	PORT=$((BASE_PORT + i - 1))

	./bin/application -n "app$i" -id "J$i" -addr localhost -port "$PORT" -web "$ROOT/web" \
	< /tmp/in_app$i > /tmp/out_app$i &
	./bin/control -n "ctl$i" -id "$i" -sites "$NB_SITES" \
	< /tmp/in_ctl$i > /tmp/out_ctl$i &
done

# Connexions :
#   out_app_i  ->  in_ctl_i                      (app -> contrôle local)
#   out_ctl_i  ->  in_app_i  +  in_ctl_{i+1}     (contrôle -> app local + anneau)
for i in $(seq 1 $NB_SITES); do
	NEXT_SITE=$(( (i % NB_SITES) + 1 ))
	cat /tmp/out_app$i > /tmp/in_ctl$i &
	tee /tmp/in_app$i < /tmp/out_ctl$i > /tmp/in_ctl$NEXT_SITE &
done

echo "URLs :"
for i in $(seq 1 $NB_SITES); do
	echo " * http://localhost:$((BASE_PORT + i - 1)) -> J$i"
done

sleep 3600
nettoyer