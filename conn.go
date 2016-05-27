package main

import (
  "github.com/gorilla/websocket"
  "github.com/shurcooL/github_flavored_markdown"
  "log"
  "net/http"
  "time"
)

const (
  writeWait = 10 * time.Second
  pongWait = 60 * time.Second
  pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
}

type connection struct {
  ws *websocket.Conn
  send chan []byte
}

func (c *connection) readPump() {
  defer func() {
    connectionHub.delete <- c
    c.ws.Close()
  }()
  c.ws.SetReadDeadline(time.Now().Add(pongWait))
  c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
  for {
    _, message, err := c.ws.ReadMessage()
    if err != nil {
      if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
        log.Printf("error: %v", err)
      }
      break
    }
    message = github_flavored_markdown.Markdown(message)
    connectionHub.broadcast <- message
  }
}

func (c *connection) write(mt int, payload []byte) error {
  c.ws.SetWriteDeadline(time.Now().Add(writeWait))
  return c.ws.WriteMessage(mt, payload)
}

func (c *connection) writePump() {
  ticker := time.NewTicker(pingPeriod)
  defer func() {
    ticker.Stop()
    c.ws.Close()
  }()
  for {
    select {
    case message, ok := <-c.send:
      if !ok {
        c.write(websocket.CloseMessage, []byte{})
        return
      }
      if err := c.write(websocket.TextMessage, message); err != nil {
        return
      }
    case <-ticker.C:
      if err := c.write(websocket.PingMessage, []byte{}); err != nil {
        return
      }
    }
  }
}

// serveWs handles websocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {
  ws, err := upgrader.Upgrade(w, r, nil)
  if err != nil {
    log.Println(err)
    return
  }
  c := &connection{send: make(chan []byte, 256), ws: ws}
  connectionHub.add <- c
  go c.writePump()
  c.readPump()
}
