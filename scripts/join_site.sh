#!/usr/bin/env bash
#
# Script pour faire rejoindre un nouveau site dans l'anneau
#
# Utilisation : ./join_site.sh <mon_id> <contact_id> [base_port]
# Exemple : ./join_site.sh 4 1

set -e

ID=$1
CONTACT=$2
BASE_PORT=${3:-4444}

if [[ -z "$ID" || -z "$CONTACT" ]]; then
	echo "Le script doit recevoir au moins 2 paramètres : <mon_id> et <contact_id>"
	exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

for bin in application control net; do
	if [[ ! -x "./bin/$bin" ]]; then
		echo "Binaire manquant : ./bin/$bin"
		echo "Compilez avec 'make build'"
		exit 1
	fi
done

# Bon pour l'instant ça c'est pas fou parce que ça suppose qu'on ajoute bien les sites en id croissant sans trou
# Mais le problème c'est qu'on doit quand même fournir cette donnée au lancement du control pour qu'il setup son horloge vectorielle
# Ou alors faudrait trouver un moyen pour que le site soit "dormant" jusqu'à ce que son parrain lui fournisse de quoi tourner ?
EXISTING=$((ID - 1))

# On créé les FIFOs de notre nouveau site
rm -f /tmp/in_app$ID /tmp/out_app$ID /tmp/in_ctl$ID /tmp/out_ctl$ID /tmp/in_net$ID /tmp/out_net$ID
mkfifo /tmp/in_app$ID /tmp/out_app$ID
mkfifo /tmp/in_ctl$ID /tmp/out_ctl$ID
mkfifo /tmp/in_net$ID /tmp/out_net$ID

# On lance les binaires Go app et ctl de notre nouveau site
PORT=$((BASE_PORT + ID - 1))
./bin/application -n "app$ID" -id "J$ID" -addr localhost -port "$PORT" -web "$ROOT/web" < /tmp/in_app$ID > /tmp/out_app$ID &
./bin/control -n "ctl$ID" -id "$ID" -sites "$EXISTING" < /tmp/in_ctl$ID > /tmp/out_ctl$ID &

# On créé les redirections stdin/stdout de l'app et du ctl
cat /tmp/out_app$ID > /tmp/in_ctl$ID &
tee /tmp/in_app$ID < /tmp/out_ctl$ID > /tmp/in_net$ID &
cat /tmp/out_net$ID > /dev/null &
TTOUT_PID=$!

# On lance le binaire GO net à la fin pour lui donner la main sur le PID de son stdout
./bin/net -n "net$ID" -id "$ID" -next "$CONTACT" -ttout "$TTOUT_PID" < /tmp/in_net$ID > /tmp/out_net$ID &

echo " * http://localhost:$PORT -> J$ID (rejoint via site $CONTACT)"
