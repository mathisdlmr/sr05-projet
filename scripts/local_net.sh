#!/usr/bin/env bash
#
# Script pour faire tourner notre système réparti en local
# Topologie : anneau de N=NB_SITES, chaque site embarque un binaire de gestion du
#             réseau (net), un controller, et une application (avec serveur web intégré).
#
# Pipeline pour chaque site i :
#   browser <--WS--> application_i (embarque le serveur HTTP/WS)
#   application_i  --stdout-->  control_i  (FIFO out_app_i -> in_ctl_i)
#   control_i      --stdout-->  application_i (local)  +  net_i (local)
#   net_i          --stdout-->  control_i + net_{i+1} (pour faire l'anneau)

NB_SITES=3
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
  killall tee 2>/dev/null
  killall cat 2>/dev/null
  rm -f /tmp/in_app* /tmp/out_app* /tmp/in_ctl* /tmp/out_ctl*
  exit 0
}
trap nettoyer INT QUIT TERM

# Pour chaque site, on créé 3 fifos (in et out) pour l'app, le contrôle et le réseau
for i in $(seq 1 $NB_SITES); do
  rm -f /tmp/in_app$i /tmp/out_app$i /tmp/in_ctl$i /tmp/out_ctl$i /tmp/in_net$i /tmp/out_net$i
  mkfifo /tmp/in_app$i  /tmp/out_app$i
  mkfifo /tmp/in_ctl$i  /tmp/out_ctl$i
  mkfifo /tmp/in_net$i  /tmp/out_net$i
done

# Lancement des processus
for i in $(seq 1 $NB_SITES); do
  PORT=$((BASE_PORT + i - 1))

  ./bin/application -n "app$i" -id "J$i" -addr localhost -port "$PORT" -web "$ROOT/web" \
    < /tmp/in_app$i > /tmp/out_app$i &
  ./bin/control -n "ctl$i" -id "$i" -sites "$NB_SITES" \
    < /tmp/in_ctl$i > /tmp/out_ctl$i &
  ./bin/net < /tmp/in_net$i > /tmp/out_net$i &
done

# Connexions :
#   out_app_i  ->  in_ctl_i                      (app -> contrôle local)
#   out_ctl_i  ->  in_app_i + in_net_i           (contrôle -> app local + réseau local)
#   out_net_i  ->  in_ctl_i  +  in_net_{i+1}     (réseau -> contrôle local + anneau)
for i in $(seq 1 $NB_SITES); do
  NEXT_SITE=$(( (i % NB_SITES) + 1 ))
  cat /tmp/out_app$i > /tmp/in_ctl$i &
  tee /tmp/in_app$i < /tmp/out_ctl$i > /tmp/in_net$i &
  tee /tmp/in_ctl$i < /tmp/out_net$i > /tmp/in_net$NEXT_SITE &
done

echo "URLs :"
for i in $(seq 1 $NB_SITES); do
  echo " * http://localhost:$((BASE_PORT + i - 1)) -> J$i"
done

sleep 3600
nettoyer
