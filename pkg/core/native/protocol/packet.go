package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Packet repräsentiert ein appleJuice Protokoll-Paket.
type Packet struct {
	ID      byte
	Payload []byte
}

// ReadPacket liest ein Paket aus einem Reader.
func ReadPacket(r io.Reader) (*Packet, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	// Debug: Zeige die ersten 4 Bytes
	// fmt.Printf("DEBUG: Header empfangen: %X\n", header)

	length := binary.BigEndian.Uint32(header)
	
	// Falls die Länge unrealistisch groß ist (> 1MB), versuchen wir es mit LittleEndian
	if length > 1024*1024 {
		length = binary.LittleEndian.Uint32(header)
	}

	if length == 0 {
		return &Packet{ID: 0, Payload: []byte{}}, nil
	}

	if length > 1024*1024 {
		return nil, fmt.Errorf("Paket zu groß: %d Bytes", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	return &Packet{
		ID:      data[0],
		Payload: data[1:],
	}, nil
}

// WritePacket schreibt ein Paket in einen Writer.
func WritePacket(w io.Writer, p *Packet) error {
	length := uint32(len(p.Payload) + 1)
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}

	if _, err := w.Write([]byte{p.ID}); err != nil {
		return err
	}

	if _, err := w.Write(p.Payload); err != nil {
		return err
	}

	return nil
}

// WriteIntPacket schreibt ein Paket mit einem 32-bit Integer als Payload.
// Laut JAR-Analyse (F:(I)V) werden Integerwerte als Big-Endian-4-Byte-Wert gesendet.
func WriteIntPacket(w io.Writer, id byte, value int32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(value))
	return WritePacket(w, &Packet{
		ID:      id,
		Payload: buf,
	})
}
