#!/usr/bin/env bash
#
# Script pour faire tourner notre système réparti en local
# Basée sur "0-old/scripts/7-ring_with_ctl.sh"

NB_SITES=3
BASE_PORT=${1:-4444}

for bin in server application control; do
  if [[ ! -x "./bin/$bin" ]]; then
    echo "Binaire manquant : ./bin/$bin"
    echo "Compilez avec `make build`"
    exit 1
  fi
done

nettoyer() {
  echo "Nettoyage..."
  killall server 2>/dev/null
  killall application 2>/dev/null
  killall control 2>/dev/null
  killall tee 2>/dev/null
  killall cat 2>/dev/null
  rm -f /tmp/in_srv* /tmp/out_srv* /tmp/in_app* /tmp/out_app* /tmp/in_ctl* /tmp/out_ctl*
  exit 0
}
trap nettoyer INT QUIT TERM

# Pour chaque site, on créé 2 fifos (in et out) pour chaque processus (server, app, control)
for i in $(seq 1 $NB_SITES); do
  mkfifo /tmp/in_srv$i  /tmp/out_srv$i
  mkfifo /tmp/in_app$i  /tmp/out_app$i
  mkfifo /tmp/in_ctl$i  /tmp/out_ctl$i
done

# Lancement des processus
for i in $(seq 1 $NB_SITES); do
  PORT=$((BASE_PORT + i - 1))

  ./bin/server -n "srv$i" -id "J$i" -addr localhost -port "$PORT" -web "./web" < /tmp/in_srv$i > /tmp/out_srv$i &
  ./bin/application -n "app$i" -id "J$i" < /tmp/in_app$i > /tmp/out_app$i &
  ./bin/control -n "ctl$i" -id "J$i" -NB_SITES "$NB_SITES" < /tmp/in_ctl$i > /tmp/out_ctl$i &
done

# Connexions entre les processus
#
# Pour chaque site i, le pipeline local est :
# * out_srv_i -> in_app_i
# * out_app_i -> in_srv_i
# * out_app_i -> in_ctl_i
# * out_ctl_i -> in_app_i
#
# Et le lien entre les centres de contrôle en anneau :
# * out_ctl_i -> in_ctl_{i+1}
for i in $(seq 1 $NB_SITES); do
  NEXT_SITE=$(( (i % NB_SITES) + 1 ))

  cat /tmp/out_srv$i > /tmp/in_app$i & # Du server vers l'app
  tee /tmp/in_srv$i < /tmp/out_app$i > /tmp/in_ctl$i & # De l'app vers le server et le control
  tee /tmp/in_app$i < /tmp/out_ctl$i > /tmp/in_ctl$NEXT_SITE & # Du control vers l'app et l'anneau avec les autres centres de contrôle
done

echo "URLs :"
for i in $(seq 1 $NB_SITES); do
  echo " * http://localhost:$((BASE_PORT + i - 1)) -> J$i"
done

sleep 3600
nettoyer