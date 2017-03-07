package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"os"
)

/*
 * Prints the chunk ID, list of Destinations and Priority
 * for any given chunk
 */

type UUID [16]byte
type CsID [16]byte

func parseHeader(chunkpath string, printall bool) {
	// fmt.Println("Parsing Header for chunk", chunkpath)
	f, _ := os.Open(chunkpath)
	defer func() {
		f.Close()
	}()
	// Read the length of outer header
	bufLen := make([]byte, 9)
	_, _ = f.Read(bufLen)
	num := binary.LittleEndian.Uint32(bufLen[1:9])

	// Read Chunk ID
	buf1 := make([]byte, 16)
	_, _ = f.Read(buf1)

	// Read hop and syncID
	unused := make([]byte, 5)
	_, _ = f.Read(unused)

	var cID [16]byte
	for i := 0; i < 16; i++ {
		cID[i] = buf1[i]
	}
	var chunkID UUID = cID
	chunkID = chunkID
	if printall {
		fmt.Println("ChunkID: ", chunkID.String())
	}

	// Read the outer header
	// Decode dcpe
	num = num - 16 - 5
	buf := make([]byte, num)
	_, _ = f.Read(buf)
	dcpe := []CsID{}
	gob.NewDecoder(bytes.NewBuffer(buf[:num-17])).Decode(&dcpe)
	prio := binary.LittleEndian.Uint32(buf[num-17 : num-9])
	prio = prio
	if printall {
		fmt.Println("Priority: ", prio)
	}
	var i int
	for i = 0; i < len(dcpe); i++ {
		fmt.Printf("%v\n", dcpe[i])
	}
	fmt.Printf("\n")
}

func (c CsID) String() string {
	var s string
	s = fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
		c[0], c[1], c[2], c[3], c[4], c[5],
		c[6], c[7], c[8], c[9], c[10], c[11],
		c[12], c[13], c[14], c[15])
	return s
}

func (b UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func main() {
	var printall bool = true
	if len(os.Args) < 2 {
		fmt.Println("Provide at least one filename to be parsed")
		return
	}
	if os.Args[1] == "--ship" {
		printall = false
		for i := 2; i < len(os.Args); i++ {
			parseHeader(os.Args[i], printall)
		}
	} else {
		for i := 1; i < len(os.Args); i++ {
			parseHeader(os.Args[i], printall)
		}
	}
}
