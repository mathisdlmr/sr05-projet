echo "----- Pour lancer avec netcat -----"

echo "1. Lancer sur un terminal le serveur"
echo "nc -l 4000 | ./prog -n B"
echo ""
echo "2. Lancer sur un autre terminal le client"
echo "./prog -n A | nc localhost 4000"