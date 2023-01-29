package rtmp

import "os"

func write(bs []byte, filename string) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	f.Write(bs)
	Log("write \"%s\" done: %d bytes", filename, len(bs))
}
