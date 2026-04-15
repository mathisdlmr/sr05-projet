package main

import (
    "flag"
    "fmt"
    "net/http"
    "time"

    "github.com/gorilla/websocket"
)

// Websocket globale
var ws *websocket.Conn = nil

// Handler HTTP simple
func do_webserver(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Serveur Go actif")
}

// Handler WebSocket
func do_websocket(w http.ResponseWriter, r *http.Request) {

    var upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }

    cnx, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("Erreur upgrade:", err)
        return
    }

    ws = cnx
    fmt.Println("WebSocket ouverte")

    for {
        _, message, err := ws.ReadMessage()
        if err != nil {
            fmt.Println("Erreur lecture:", err)
            ws = nil
            return
        }

        fmt.Println("Reçu :", string(message))
    }
}

// Envoi vers client
func ws_send(msg string) {
    if ws == nil {
        fmt.Println("ws_send : websocket non ouverte")
        return
    }

    err := ws.WriteMessage(websocket.TextMessage, []byte(msg))
    if err != nil {
        fmt.Println("Erreur envoi:", err)
    } else {
        fmt.Println("Envoyé :", msg)
    }
}

// Envoi périodique
func do_send() {
    for {
        ws_send("Hello depuis le serveur")
        time.Sleep(4 * time.Second)
    }
}

func main() {

    var port = flag.String("port", "4444", "port")
    var addr = flag.String("addr", "localhost", "adresse")

    flag.Parse()

    http.HandleFunc("/", do_webserver)
    http.HandleFunc("/ws", do_websocket)

    go do_send()

    fmt.Println("Serveur lancé sur", *addr+":"+*port)
    http.ListenAndServe(*addr+":"+*port, nil)
}