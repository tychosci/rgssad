package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"unsafe"
)

type Rgssad struct {
	input *os.File
	items []*Item
}

type Item struct {
	filename string
	filesize uint32
	magickey uint32
	offset   int64
}

func isValidHeader(header []byte) bool {
	return (header[0] == 'R' || header[1] == 'G' ||
		header[2] == 'S' || header[3] == 'S' ||
		header[4] == 'A' || header[5] == 'D')
}

func Extract(filename string) (*Rgssad, error) {
	in, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	var items []*Item

	var header [8]byte
	in.Read(header[0:len(header)])
	// Checks whether input file is rgssad format or not.
	if !isValidHeader(header[0:8]) {
		return nil, fmt.Errorf("rgssad: %s is not a rgssad file", filename)
	}

	magickey := uint32(0xdeadcafe)

	decrypt_byte := func(v byte) byte {
		w := v ^ byte(magickey&0xff)
		magickey = magickey*7 + 3
		return w
	}
	decrypt_u32 := func(v uint32) uint32 {
		w := v ^ uint32(magickey&0xffffffff)
		magickey = magickey*7 + 3
		return w
	}

	for {
		var filenamesize, filesize uint32

		// decrypt filenamesize
		err := binary.Read(in, binary.LittleEndian, &filenamesize)
		if err == io.EOF {
			break
		}
		filenamesize = decrypt_u32(filenamesize)

		// decrypt filename
		filename_buf := make([]byte, filenamesize)
		in.Read(filename_buf[0:filenamesize])
		for i := uint32(0); i < filenamesize; i++ {
			filename_buf[i] = decrypt_byte(filename_buf[i])
			if filename_buf[i] == byte('\\') {
				filename_buf[i] = byte('/')
			}
		}
		filename := string(filename_buf)

		// decrypt filesize
		binary.Read(in, binary.LittleEndian, &filesize)
		filesize = decrypt_u32(filesize)

		// current offset
		offset, _ := in.Seek(int64(0), os.SEEK_CUR)
		item := &Item{
			filename: filename,
			filesize: filesize,
			magickey: magickey,
			offset:   offset,
		}

		// add item
		items = append(items, item)

		// skip data section
		in.Seek(int64(filesize), os.SEEK_CUR)
	}

	return &Rgssad{input: in, items: items}, nil
}

func (self *Rgssad) Show() {
	for _, item := range self.items {
		fmt.Println("filename:", item.filename)
		fmt.Println("  filesize =", item.filesize)
		fmt.Println("  magickey =", item.magickey)
	}
}

func (self *Rgssad) Save() {
	in := self.input

	for _, item := range self.items {
		fmt.Printf("[save]: %s (size %v)\n", item.filename, item.filesize)

		// Creates missing directories in item.filename
		dirname, _ := path.Split(item.filename)
		os.MkdirAll(dirname, 0x1e8) // rwxr-xr-x

		// Creates output file (rw-r--r--) for the decrypted data
		out, err := os.OpenFile(item.filename, os.O_WRONLY|os.O_CREATE, 0x180)
		if err != nil {
			fmt.Println("[fail]:", item.filename)
			fmt.Println("  reason =", err)
			return
		}
		// Jumps to item's data section
		in.Seek(item.offset, os.SEEK_SET)

		saveItem(in, out, item)

		out.Close()
	}
}

func saveItem(in, out *os.File, item *Item) {
	var buf [1024]byte
	magickey := item.magickey
	leftsize := item.filesize

	decrypt_cb := func(b []byte) {
		for i := 0; i < len(b); i += 4 {
			bufptr := unsafe.Pointer(&buf[i])
			*(*uint32)(bufptr) ^= magickey
			magickey = magickey*7 + 3
		}
	}

	for leftsize > 1024 {
		in.Read(buf[0:1024])
		decrypt_cb(buf[0:1024])
		out.Write(buf[0:1024])
		leftsize -= 1024
	}

	in.Read(buf[0:leftsize])
	decrypt_cb(buf[0:leftsize])
	out.Write(buf[0:leftsize])
}

func (self *Rgssad) Close() {
	self.input.Close()
}
