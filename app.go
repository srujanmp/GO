// websockets.go
package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	http.HandleFunc("/socket", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		for{
			_, msg, err:= conn.ReadMessage() //recieves msg
			if err != nil {return}

			fmt.Println(string(msg),"recieved")  //prints msg to console

			err=conn.WriteMessage(websocket.TextMessage, []byte("pong")) //sends pong to frontend
			if err != nil {return}
			fmt.Println("pong sent")
			
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	http.ListenAndServe(":8080", nil)
}
