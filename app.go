package main

import (
	"fmt"
	"io"
	"net/http"
	"os"   //for file
	"time" //for timer

	"github.com/gorilla/websocket" //for websocket
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	ticker := time.NewTicker(1 * time.Millisecond) //20ms
	defer ticker.Stop()                            // Stop ticker when the server shuts down

	http.HandleFunc("/socket", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			// optional: log the error
			return
		}
		audioFile, _ := os.Open("audio.wav")
		buffer := make([]byte, 160) // 160 bytes per chunk

		// Channel to communicate state
		stateChan := make(chan string, 1)

		// Goroutine to read messages from WebSocket
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					break
				}

				fmt.Println(string(msg))
				stateChan <- string(msg)
			}
		}()

		currentState := "pause" // Default state
		audioFile.Seek(0, 0)
		// Main loop to handle ticker and state
		for range ticker.C {
			// Read a chunk of 160 bytes from the file

			select {
			case newState := <-stateChan: // Check if state is updated
				currentState = newState
			default:
				//continue
			}

			if currentState == "pause" {
				continue
			}

			n, err := audioFile.Read(buffer)
			if err == io.EOF {
				// Reached end of file
				fmt.Println("End of audio file reached.")
				audioFile.Seek(0, 0) // Go back to the start of the file
				conn.WriteMessage(websocket.TextMessage, []byte("eof"))
				continue
			}
			if err != nil {
				fmt.Println("Error reading file:", err)

			}

			conn.WriteMessage(websocket.BinaryMessage, buffer[:n])

		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	http.ListenAndServe(":8080", nil)
}
