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
	ReadBufferSize:  160,
	WriteBufferSize: 160,
}

func main() {
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop() // Stop ticker when the server shuts down

	http.HandleFunc("/socket", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		audioFile, _ := os.Open("audio.wav")
		buffer := make([]byte, 160) // 160 bytes per chunk

		// Channel to communicate state
		stateChan := make(chan string, 1)

		// Goroutine to read messages from WebSocket
		go func() {
			for {
				_, msg, _ := conn.ReadMessage()
				stateChan <- string(msg)
			}
		}()

		currentState := "pause" // Default state

		// Main loop to handle ticker and state
		for t := range ticker.C {
			// Read a chunk of 160 bytes from the file
			n, err := audioFile.Read(buffer)
			if err == io.EOF {
				// Reached end of file, loop back to start or end the audio
				fmt.Println("End of audio file reached.")
				audioFile.Seek(0, 0) // Go back to the start of the file
				continue
			}
			if err != nil {
				fmt.Println("Error reading file:", err)
				break
			}

			select {
			case newState := <-stateChan: // Check if state is updated
				currentState = newState
			default:
				//continue
			}

			fmt.Println(t)
			if currentState == "pause" {
				continue
			}
			conn.WriteMessage(websocket.BinaryMessage, buffer[:n])

		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	http.ListenAndServe(":8080", nil)
}
