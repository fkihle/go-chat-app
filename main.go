package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"unicode"

	"github.com/gorilla/websocket"
)

// ######################################################################
// struct: Chatter
// ######################################################################
type Chatter struct {
	conn     *websocket.Conn
	username string
	// strikes int
}

var (
	chatters = make(map[*Chatter]bool)
	mutex    = &sync.Mutex{}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Add CheckOrigin function if necessary for CORS
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ######################################################################
// function: cleanInput()
// ######################################################################
func cleanInput(input string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, input)
}

// ######################################################################
// function: handleConnection()
// ######################################################################
func handleConnection(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error: ", err)
		return
	}
	defer ws.Close()

	// Create a new chatter and add to the chatters map
	chatter := &Chatter{conn: ws, username: "dorkiBallz"}
	mutex.Lock()
	chatters[chatter] = true
	mutex.Unlock()

	ws.WriteMessage(websocket.TextMessage, []byte("Velkommen til chat.\n"))
	ws.WriteMessage(websocket.TextMessage, []byte("Bytt brukernavn med: /u <ditt_brukernavn>\n"))
	ws.WriteMessage(websocket.TextMessage, []byte("Forlat chat med: /q\n"))
	// defer closing connection and deleting chatters til end of function
	defer func() {
		mutex.Lock()
		delete(chatters, chatter)
		mutex.Unlock()
	}()

	for {
		messageType, bytemessage, err := ws.ReadMessage()
		if err != nil {
			log.Println("Read error: ", err)
			break
		}

		// HANDLE THE MESSAGE
		// For example, broadcast the message to other connected clients
		// Make sure to handle different types of messages (text, binary, etc.)
		if messageType == websocket.TextMessage {
			message := string(bytemessage)

			if strings.HasPrefix(message, "/u ") {
				// Set the username
				chatter.username = strings.TrimSpace(strings.TrimPrefix(message, "/u "))
				ws.WriteMessage(websocket.TextMessage, []byte("Username set to "+chatter.username+"\n"))

			} else if strings.HasPrefix(message, "/q") {
				fmt.Printf("User %s has disconnected.\n", chatter.username)
				break // exit the loop to close the connection

			} else {
				// Broadcast the message
				broadcast(fmt.Sprintf("%s: %s", chatter.username, message), chatter)
				ws.WriteMessage(websocket.TextMessage, []byte(chatter.username+": "+message+"\n"))
			}
		} else if messageType == websocket.BinaryMessage {
			broadcast(fmt.Sprintf("%s has entered a binary message. For shame!", chatter.username), nil)
			fmt.Printf("User %s has entered a binary message. For shame!\n", chatter.username)
		}

	}

	// Once the loop exits, the client has disconnected
	broadcast(fmt.Sprintf("%s has left the chat.", chatter.username), nil)
}

// ######################################################################
// function: broadcast()
// ######################################################################
func broadcast(message string, sender *Chatter) {
	mutex.Lock()
	defer mutex.Unlock()
	for chatter := range chatters {
		if sender == nil || chatter != sender {
			err := chatter.conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}
		}
	}
}

// ######################################################################
// function: main()
// ######################################################################
func main() {
	// Set up WebSocket route
	http.HandleFunc("/ws", handleConnection)

	// Serve static files from a directory
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	fmt.Println("WebSocket server started on port 6969")
	log.Fatal(http.ListenAndServe("https://go-chat-app-0192827617c6.herokuapp.com", nil))
}
