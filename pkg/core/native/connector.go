package native

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/applejuicenetz/ajnx/pkg/core/native/protocol"
)

// ConnectToServer baut eine Verbindung zu einem appleJuice Server auf.
func (c *NativeCore) ConnectToServer(ctx context.Context, host string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)
	log.Printf("NativeCore: Verbinde mit Server %s...", address)

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Verbindung fehlgeschlagen: %v", err)
	}

	c.mu.Lock()
	if c.currentConn != nil {
		c.currentConn.Close()
	}
	c.currentConn = conn

	// Sofort als "Verbinde..." markieren für bessere UX
	for i, s := range c.servers {
		if s.Host == host && s.Port == port {
			c.servers[i].Status = "Verbinde..."
			c.info.Network.ConnectedWithServer = "Verbinde mit " + s.Name + "..."
		}
	}
	c.mu.Unlock()

	// Handshake starten (korrektes Protokoll laut JAR-Analyse)
	if err := c.performHandshake(conn); err != nil {
		conn.Close()
		c.mu.Lock()
		if c.currentConn == conn {
			c.currentConn = nil
		}
		c.mu.Unlock()
		return fmt.Errorf("handshake fehlgeschlagen: %v", err)
	}

	// Hintergrund-Loop für Pakete
	go c.handleServerConnection(conn, host, port)

	return nil
}

// performHandshake führt den appleJuice-Handshake durch.
// Laut JAR-Bytecode (G/A.class, G/E.class und B.class):
//  1. Client -> Server: "ajprot\r\n" (Roh-Bytes)
//  2. Client -> Server: 145 (uint16, Big-Endian, Roh-Bytes)
//  3. Server -> Client: Server-Version (uint16, Big-Endian, Roh-Bytes)
//  4. AB HIER: Standard-Paketprotokoll (Length [uint32] + ID [byte] + Payload)
func (c *NativeCore) performHandshake(conn net.Conn) error {
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetDeadline(time.Time{})

	// Schritt 1: "ajprot\r\n" (Roh)
	log.Printf("NativeCore: Sende ajprot...")
	if _, err := conn.Write([]byte("ajprot\r\n")); err != nil {
		return fmt.Errorf("fehler beim Senden von ajprot: %v", err)
	}

	// Schritt 2: Version 145 (uint16 BE, Roh)
	log.Printf("NativeCore: Sende Version 145...")
	verBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(verBuf, 145)
	if _, err := conn.Write(verBuf); err != nil {
		return fmt.Errorf("fehler beim Senden der Version: %v", err)
	}

	// Schritt 3: Server-Version lesen (uint16 BE, Roh)
	log.Printf("NativeCore: Warte auf Server-Version...")
	if _, err := io.ReadFull(conn, verBuf); err != nil {
		return fmt.Errorf("fehler beim Lesen der Server-Version: %v", err)
	}
	srvVer := binary.BigEndian.Uint16(verBuf)
	log.Printf("NativeCore: Server-Version erhalten: %d", srvVer)

	// Schritt 4: Login-Paket (Ab hier Standard-Paket-Format)
	return c.sendLogin(conn)
}

func (c *NativeCore) sendLogin(conn net.Conn) error {
	c.mu.RLock()
	nick := c.getSetting("nick")
	if nick == "" {
		nick = "AJNX-User"
	}
	port := c.getSetting("port")
	if port == "" {
		port = "9000"
	}
	xmlPort := c.getSetting("xmlport")
	if xmlPort == "" {
		xmlPort = "9859"
	}
	c.mu.RUnlock()

	// appleJuice Login (ID 1)
	// Format: Nickname|TCPPort|Version|XMLPort|
	payload := fmt.Sprintf("%s|%s|149|%s|", nick, port, xmlPort)
	p := &protocol.Packet{
		ID:      1,
		Payload: []byte(payload),
	}
	log.Printf("NativeCore: Sende Login-Paket: %q", payload)
	return protocol.WritePacket(conn, p)
}

func (c *NativeCore) handleServerConnection(conn net.Conn, targetHost string, targetPort int) {
	defer func() {
		conn.Close()
		c.mu.Lock()
		if c.currentConn == conn {
			c.currentConn = nil
			c.info.Network.ConnectedWithServer = ""
			for i := range c.servers {
				c.servers[i].Connected = false
				c.servers[i].Status = "Getrennt"
			}
		}
		c.mu.Unlock()
		log.Printf("NativeCore: Verbindung zu %s:%d beendet.", targetHost, targetPort)
	}()
	log.Printf("NativeCore: Server-Verbindung aktiv (%s:%d). Warte auf Pakete...", targetHost, targetPort)

	for {
		p, err := protocol.ReadPacket(conn)
		if err != nil {
			log.Printf("NativeCore: Server-Verbindung verloren: %v", err)
			return
		}

		switch p.ID {
		case 2: // Welcome
			log.Printf("NativeCore: Willkommen vom Server: %q", string(p.Payload))
			c.mu.Lock()
			for i, s := range c.servers {
				if s.Host == targetHost && s.Port == targetPort {
					c.servers[i].Connected = true
					c.servers[i].Status = "Verbunden"
					c.info.Network.ConnectedWithServer = s.Name
					log.Printf("NativeCore: Verbunden mit %s", s.Name)
				} else {
					c.servers[i].Connected = false
					c.servers[i].Status = "Getrennt"
				}
			}
			c.mu.Unlock()

		case 3: // Ping -> Pong
			_ = protocol.WritePacket(conn, &protocol.Packet{ID: 4, Payload: []byte{}})
			log.Printf("NativeCore: Ping/Pong.")

		default:
			log.Printf("NativeCore: Paket empfangen: ID=%d (%d Bytes)", p.ID, len(p.Payload))
		}
	}
}
