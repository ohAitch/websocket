package chat

import (
	"log"
	"net/http"

	"code.google.com/p/go.net/websocket"
)

// Chat server.
type Server struct {
	pattern   string
	messages  []*Message
	clients   map[int]*Client
	doneCh    chan bool
	errCh     chan error
        deferred  chan *func
}

// Create new chat server.
func NewServer(pattern string) *Server {
	messages := []*Message{}
	clients := make(map[int]*Client)
	doneCh := make(chan bool)
	errCh := make(chan error)

	return &Server{
		pattern,
		messages,
		clients,
		doneCh,
		errCh,
	}
}

func (s *Server) Add(c *Client) {
        s.deferred <- func() {
		log.Println("Added new client")
		s.clients[c.id] = c
		log.Println("Now", len(s.clients), "clients connected.")
		s.sendPastMessages(c)
	}
}

func (s *Server) Del(c *Client) {
        s.deferred <- func() {
		log.Println("Delete client")
		delete(s.clients, c.id)
	}
}

func (s *Server) SendAll(msg *Message) {
        s.deferred <- func() {
		log.Println("Send all:", msg)
		s.messages = append(s.messages, msg)
		s.sendAll(msg)
	}
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
}

func (s *Server) sendPastMessages(c *Client) {
	for _, msg := range s.messages {
		c.Write(msg)
	}
}

func (s *Server) sendAll(msg *Message) {
	for _, c := range s.clients {
		c.Write(msg)
	}
}

// Listen and serve.
// It serves client connection and broadcast request.
func (s *Server) Listen() {

	log.Println("Listening server...")

	// websocket handler
	onConnected := func(ws *websocket.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		client := NewClient(ws, s)
		s.Add(client)
		client.Listen()
	}
	http.Handle(s.pattern, websocket.Handler(onConnected))
	log.Println("Created handler")

	for {
		select {
 
		// Add new a client
		case f := <-s.deferred:
			f()

		case err := <-s.errCh:
			log.Println("Error:", err.Error())

		case <-s.doneCh:
			return
		}
	}
}
