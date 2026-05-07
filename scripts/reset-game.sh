#!/usr/bin/env bash
# Réinitialise une partie en cours sur Kube
#
# Stratégie : scaler à 0 -> tous les pods sont kill -> toutes les FIFOs se ferment
# Les fichiers FIFO sur le PVC restent en place (les init containers les réutilisent au redémarrage)

set -euo pipefail

NAMESPACE="${1:-werewolf}"
REPLICAS="${2:-}"

if [[ -z "$REPLICAS" ]]; then
  REPLICAS=$(kubectl get statefulset werewolf -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "")
  if [[ -z "$REPLICAS" ]]; then
    echo "Impossible de lire le nombre de replicas. Précisez-le en 2e argument."
    echo "Usage: $0 [namespace] [nb_joueurs]"
    exit 1
  fi
fi

echo "Reset de la partie (namespace: $NAMESPACE, joueurs: $REPLICAS)"

kubectl scale statefulset werewolf -n "$NAMESPACE" --replicas=0
kubectl wait --for=delete pod -l app=werewolf -n "$NAMESPACE" --timeout=60s 2>/dev/null || true
kubectl scale statefulset werewolf -n "$NAMESPACE" --replicas="$REPLICAS"

echo "Prêt, les joueurs peuvent se reconnecter sur leurs URLs respectives."
