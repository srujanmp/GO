package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WebSocketMessage struct {
	Type string                    `json:"type"`
	SDP  webrtc.SessionDescription `json:"sdp"`
}

type AudioServer struct {
	mutex          sync.Mutex
	isPlaying      bool
	audioFile      *os.File
	peerConnection *webrtc.PeerConnection
}

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/websocket", handleWebSocket)

	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer c.Close()

	audioServer := &AudioServer{
		isPlaying: false,
	}

	defer func() {
		if audioServer.audioFile != nil {
			audioServer.audioFile.Close()
		}
	}()

	// Set media engine with proper codecs
	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     "audio/PCMU",
			ClockRate:    8000,
			Channels:     1,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 0,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		log.Printf("Error registering codec: %v", err)
		return
	}

	// Create API with media engine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))

	// WebRTC configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	// Create peer connection using the API
	peerConnection, _ := api.NewPeerConnection(config)
	defer peerConnection.Close()
	audioServer.peerConnection = peerConnection

	// Create audio track with specific parameters
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:    "audio/PCMU",
			ClockRate:   8000,
			Channels:    1,
			SDPFmtpLine: "",
		},
		"audio",
		"audio-stream",
	)
	if err != nil {
		log.Printf("Failed to create audio track: %v", err)
		return
	}

	sender, err := peerConnection.AddTrack(audioTrack)
	if err != nil {
		log.Printf("Failed to add track: %v", err)
		return
	}

	// Handle RTCP packets
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := sender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	// Handle WebSocket messages
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		var message WebSocketMessage
		if err := json.Unmarshal(msg, &message); err != nil {
			// If not JSON, treat as plain text command
			command := string(msg)
			switch command {
			case "PLAY":
				audioServer.mutex.Lock()
				audioServer.isPlaying = true
				audioServer.mutex.Unlock()

				// Open audio file
				audioFile, err := os.Open("audio.wav")
				if err != nil {
					log.Printf("Failed to open audio file: %v", err)
					continue
				}
				audioServer.audioFile = audioFile

				// Skip WAV header (44 bytes)
				audioFile.Seek(44, 0)

				log.Println("Starting audio stream...")

				// Start streaming audio
				go func() {
					buffer := make([]byte, 160)
					ticker := time.NewTicker(20 * time.Millisecond)
					defer ticker.Stop()
					samplesRead := 0

					for range ticker.C {
						audioServer.mutex.Lock()
						if !audioServer.isPlaying {
							audioServer.mutex.Unlock()
							break
						}
						audioServer.mutex.Unlock()

						n, err := audioFile.Read(buffer)
						if err == io.EOF {
							audioFile.Seek(44, 0)
							continue
						}
						if err != nil {
							log.Printf("Failed to read audio file: %v", err)
							break
						}

						samplesRead++
						if samplesRead%50 == 0 { // Log every 50 samples
							log.Printf("Sent %d samples, last sample size: %d bytes", samplesRead, n)
						}

						err = audioTrack.WriteSample(media.Sample{
							Data:     buffer[:n],
							Duration: 20 * time.Millisecond,
						})
						if err != nil {
							log.Printf("Failed to write sample: %v", err)
							break
						}
					}
					log.Println("Audio streaming stopped")
				}()

			case "PAUSE":
				audioServer.mutex.Lock()
				audioServer.isPlaying = false
				// if audioServer.audioFile != nil {
				// 	audioServer.audioFile.Close()
				// 	audioServer.audioFile = nil
				// }
				audioServer.mutex.Unlock()
				log.Println("Audio streaming paused")
			}
			continue
		}

		// Handle WebRTC signaling
		switch message.Type {
		case "offer":
			log.Println("Received offer, creating answer...")
			err = peerConnection.SetRemoteDescription(message.SDP)
			if err != nil {
				log.Printf("Error setting remote description: %v", err)
				continue
			}

			// Create answer
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				log.Printf("Error creating answer: %v", err)
				continue
			}

			// Set local description
			err = peerConnection.SetLocalDescription(answer)
			if err != nil {
				log.Printf("Error setting local description: %v", err)
				continue
			}
			log.Println("OFFER:")
			log.Println(message.SDP)
			log.Println("ANSWER:")
			log.Println(answer)
			log.Println("Sending answer to client...")
			// Send answer back to client
			response := WebSocketMessage{
				Type: "answer",
				SDP:  *peerConnection.LocalDescription(),
			}

			err = c.WriteJSON(response)
			if err != nil {
				log.Printf("Error sending answer: %v", err)
				continue
			}
		}
	}
}
