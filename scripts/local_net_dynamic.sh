#!/usr/bin/env bash
#
# Script pour faire tourner notre système réparti en local
# Topologie : anneau de N=NB_SITES, chaque site embarque un binaire de gestion du
#             réseau (net), un controller, et une application (avec serveur web intégré).
#
# Pipeline pour chaque site i :
#   browser <--WS--> application_i
#   application_i  --stdout-->  control_i
#   control_i      --stdout-->  application_i + net_i
#   net_i          --stdout-->  control_i local + net_{voisins}
#
# Format des messages addLink / removeLink :
#   /=type=net/=action=addLink/=sender=<id>/=target=<j>
#   /=type=net/=action=removeLink/=sender=<id>/=target=<j>
#
# Chaque out_net_i est lu par UN SEUL processus tee qui écrit vers :
#   - in_ctl_i                     (toujours, pour communiquer avec le controller local)
#   - in_net_{voisins}             (dynamique, listées dans /tmp/net_dests_$i)
#   - CMD_PIPE (stdout du tee)     (pour intercepter les requêtes de changement de topologie)
# Pour modifier les liens d'un site, on kill ce tee et on en relance un nouveau

NB_SITES=3
BASE_PORT=${1:-4444}

# Se placer à la racine du projet quel que soit le cwd d'appel
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CMD_PIPE=/tmp/net_cmd
rm -f $CMD_PIPE
mkfifo $CMD_PIPE

for bin in application control net; do
    if [[ ! -x "./bin/$bin" ]]; then
        echo "Binaire manquant : ./bin/$bin"
        echo "Compilez avec 'make build'"
        exit 1
    fi
done

# Nettoyage de tous les processus à l'interruption
nettoyer() {
    echo "Nettoyage..."
    killall application 2>/dev/null
    killall control 2>/dev/null
    killall net 2>/dev/null
    killall tee 2>/dev/null
    killall cat 2>/dev/null
    rm -f /tmp/in_app* /tmp/out_app* /tmp/in_ctl* /tmp/out_ctl* /tmp/in_net* /tmp/out_net*
    rm -f /tmp/net_router_*.pid /tmp/ctl_router_*.pid /tmp/app_router_*.pid
    rm -f /tmp/net_dests_*
    rm -f $CMD_PIPE
    exit 0
}
trap nettoyer INT QUIT TERM

# (re)démarre le processus tee qui route les messages sortant d'un out_net_i
start_net_router() {
    local id=$1
    local pid_file="/tmp/net_router_$id.pid"

    # 1. On kill le routeur existant
    if [ -f "$pid_file" ]; then
        kill "$(cat "$pid_file")" 2>/dev/null
        rm "$pid_file"
    fi

    # 2. On reconstruit la liste des destinations (in_ctl_i + in_net_{voisins})
    local dests=("/tmp/in_ctl$id")
    if [ -f "/tmp/net_dests_$id" ]; then
        while IFS= read -r dest; do
            [[ -n "$dest" ]] && dests+=("$dest")
        done < "/tmp/net_dests_$id"
    fi

    # 3. On recréé un tee qui écrit vers toutes les destinations ET envoie une 
    # copie sur notre CMD_PIPE pour que process_commands puisse lire les messages de contrôle
    tee "${dests[@]}" < "/tmp/out_net$id" > "$CMD_PIPE" &
    echo $! > "$pid_file"
}

# ajouute dst à la liste des destinations de src et redémarre le routeur de src
add_link() {
    local src=$1
    local dst=$2
    local dest="/tmp/in_net$dst"
    local dests_file="/tmp/net_dests_$src"

    # On évite les duplicats
    if grep -qxF "$dest" "$dests_file" 2>/dev/null; then
        echo "[ADD] $src -> $dst : lien déjà existant, ignoré"
        return
    fi

    echo "$dest" >> "$dests_file"
    start_net_router "$src"
    echo "[ADD] $src -> $dst"
}

# retire dst de la liste des destinations de src et redémarre le routeur de src
remove_link() {
    local src=$1
    local dst=$2
    local dest="/tmp/in_net$dst"
    local dests_file="/tmp/net_dests_$src"

    if [ ! -f "$dests_file" ]; then
        echo "[DEL] $src -> $dst : lien introuvable"
        return
    fi

    # Filtre la destination à retirer
    grep -vxF "$dest" "$dests_file" > "${dests_file}.tmp" && mv "${dests_file}.tmp" "$dests_file"
    start_net_router "$src"
    echo "[DEL] $src -> $dst"
}

# lit en continu depuis CMD_PIPE et réagit aux messages addLink/removeLink
# ne parse que les messages au format /=type=net/=action=addLink/=sender=<id>/=target=<j>
process_commands() {
    while IFS= read -r line; do
        [[ "$line" != *"/=type=net"* ]] && continue

        action=$(echo "$line" | grep -oP '(?<=/=action=)[^/]+')
        sender=$(echo "$line" | grep -oP '(?<=/=sender=)[^/]+')
        target=$(echo "$line"  | grep -oP '(?<=/=target=)[^/]+')

        case "$action" in
            addLink)    add_link    "$sender" "$target" ;;
            removeLink) remove_link "$sender" "$target" ;;
        esac
    done < "$CMD_PIPE"
}

# Pour chaque site, on créé 3 fifos (in et out) pour l'app, le contrôle et le réseau ("net")
for i in $(seq 1 $NB_SITES); do
    rm -f /tmp/in_app$i /tmp/out_app$i /tmp/in_ctl$i /tmp/out_ctl$i /tmp/in_net$i /tmp/out_net$i
    mkfifo /tmp/in_app$i  /tmp/out_app$i
    mkfifo /tmp/in_ctl$i  /tmp/out_ctl$i
    mkfifo /tmp/in_net$i  /tmp/out_net$i
done

# Lancement des processus
for i in $(seq 1 $NB_SITES); do
    PORT=$((BASE_PORT + i - 1))
    ./bin/application -n "app$i" -id "J$i" -addr localhost -port "$PORT" -web "$ROOT/web" < /tmp/in_app$i > /tmp/out_app$i &
    ./bin/control -n "ctl$i" -id "$i" -sites "$NB_SITES" < /tmp/in_ctl$i > /tmp/out_ctl$i &
    ./bin/net -n "net$i" -id "$i" < /tmp/in_net$i > /tmp/out_net$i &
done

# Connexions statiques pour les messages de l'application et du controller
for i in $(seq 1 $NB_SITES); do
    cat /tmp/out_app$i > /tmp/in_ctl$i &
    echo $! > /tmp/app_router_$i.pid

    tee /tmp/in_net$i < /tmp/out_ctl$i > /tmp/in_app$i &
    echo $! > /tmp/ctl_router_$i.pid
done

# Connexions dynamiques pour out_net_i (Topologie initiale en anneau : out_net_i -> in_ctl_i + in_net_{i+1})
for i in $(seq 1 $NB_SITES); do
    NEXT_SITE=$(( (i % NB_SITES) + 1 ))
    echo "/tmp/in_net$NEXT_SITE" > /tmp/net_dests_$i
    start_net_router "$i"
done

process_commands &

echo "URLs :"
for i in $(seq 1 $NB_SITES); do
    echo " * http://localhost:$((BASE_PORT + i - 1)) -> J$i"
done

sleep 3600
nettoyer