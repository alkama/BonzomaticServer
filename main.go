package main

import (
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/gorilla/websocket"
)

type errorCode string

func (e errorCode) String() string {
	return string(e)
}

type bsError struct {
	code errorCode
	expr string
}

func (e *bsError) Error() string {
	return "error: " + e.code.String() + ": `" + e.expr + "`"
}

const (
	errWrongPath errorCode = "Wrong path"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 64 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type client struct {
	referee *referee
	room    string
	nick    string
	conn    *websocket.Conn
	send    chan []byte
}

type message struct {
	room string
	nick string
	msg  []byte
}

func (c *client) readFromConnectionPump() {
	defer func() {
		c.referee.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		if c.nick != "" {
			c.referee.broadcast <- message{room: c.room, nick: c.nick, msg: msg}
		}
	}
}

func (c *client) writeToConnectionPump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The referee closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type referee struct {
	register   chan *client
	unregister chan *client
	clients    map[*client]bool
	broadcast  chan message
}

func newReferee() *referee {
	return &referee{
		register:   make(chan *client),
		unregister: make(chan *client),
		clients:    make(map[*client]bool),
		broadcast:  make(chan message),
	}
}

func (r *referee) run() {
	for {
		select {
		case client := <-r.register:
			log.Printf("+++client: %p\n", client)
			r.clients[client] = true

		case client := <-r.unregister:
			if _, ok := r.clients[client]; ok {
				log.Printf("---client: %p\n", client)
				delete(r.clients, client)
				close(client.send)
			}

		case message := <-r.broadcast:
			for client := range r.clients {
				if client.room == message.room && (client.nick == message.nick || client.nick == "") {
					select {
					case client.send <- message.msg:
					default:
						close(client.send)
						delete(r.clients, client)
					}
				}
			}
		}
	}
}

func getRoomAndNick(url string) (*string, *string, error) {
	re, err := regexp.Compile(`^/([A-Za-z0-9_]{3,64})/([A-Za-z0-9_]{0,16})$`)
	if err != nil {
		return nil, nil, err
	}
	match := re.FindStringSubmatch(url)
	if len(match) < 2 {
		return nil, nil, &bsError{code: errWrongPath, expr: "Wrong room name and/or nickname."}
	}
	return &match[1], &match[2], nil
}

func main() {
	log.Println("Launching Bonzomatic server.")

	referee := newReferee()
	go referee.run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		room, nick, err := getRoomAndNick(r.URL.Path)
		if err != nil {
			log.Println(err)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("Client connected to room: '" + *room + "' with nick: '" + *nick + "'")
		client := &client{referee: referee, room: *room, nick: *nick, conn: conn, send: make(chan []byte, 256)}
		client.referee.register <- client
		go client.writeToConnectionPump()
		go client.readFromConnectionPump()
	})

	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
