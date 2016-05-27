package main

type hub struct {
  connections map[*connection]bool
  broadcast chan []byte
  add chan *connection
  delete chan *connection
  currentHTML []byte
}

var connectionHub = hub{
  connections: make(map[*connection]bool),
  broadcast:   make(chan []byte),
  add:         make(chan *connection),
  delete:      make(chan *connection),
}

func (connectionHub *hub) run() {
  for {
    select {
    case c := <- connectionHub.add:
      connectionHub.connections[c] = true
      c.send <- connectionHub.currentHTML
    case c := <- connectionHub.delete:
      if _, ok := connectionHub.connections[c];
      ok {
        delete(connectionHub.connections, c)
        close(c.send)
      }
    case m := <- connectionHub.broadcast:
      connectionHub.currentHTML = m
      for c := range connectionHub.connections {
        select {
        case c.send <- m:
        default:
          close(c.send)
          delete(connectionHub.connections, c)
        }
      }
    }
  }
}
