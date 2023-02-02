# rtmp
rtmp implement with golang.


## Server 示例

```go
package main

import (
	"fmt"
	"log"

	"github.com/chenyj/rtmp"
	"github.com/chenyj/rtmp/encoding/av"
)

func main() {
	streams := map[string]rtmp.Streamer{}

	rtmp.HandleCommand(rtmp.CMD_FCUNPUBLISH, func(w rtmp.MessageWriter, r *rtmp.Request) error {
		s, ok := streams[r.StreamPath]
		if !ok {
			return nil
		}
		s.Write(nil)
		return nil
	})

	rtmp.HandleCommand(rtmp.CMD_PLAY, func(w rtmp.MessageWriter, r *rtmp.Request) error {
		s, ok := streams[r.StreamPath]
		if !ok {
			return rtmp.ResponsePlay(w, false, "stream not found")
		}
		err := rtmp.ResponsePlay(w, true, "")
		if err != nil {
			return err
		}

		go func(it rtmp.Iterator) {
			for {
				p, err := it.Next()
				if err != nil {
					break
				}
				err = w.WriteMessage(rtmp.NewMessage(p))
				if err != nil {
					break
				}
			}
			fmt.Println("播放结束")
		}(s.Iterator())

		return nil
	})

	rtmp.HandleData(func(app, path string, p *av.Packet) error {
		s, ok := streams[path]
		if !ok {
			s = rtmp.NewStream(3000)
			streams[path] = s
		}
		s.Write(p)
		return nil
	})

	err := rtmp.ListenAndServe("", nil)
	log.Fatal(err)
}

```

- 推流：

`ffmpeg -re -stream_loop -1 -i trailer.mp4 -codec copy -f flv rtmp://localhost/live/test`

- 拉流：

`ffplay -autoexit rtmp://localhost/live/test`


# Client 示例

```go
package main

import (
	"fmt"

	"github.com/chenyj/rtmp/encoding/flv"

	"github.com/chenyj/rtmp"
)

func main() {
	cli := rtmp.NewClient()
	defer cli.Close()

	// rtmp://loaclhost:1935/live/test
	cli.Dail(":1935").Handshake().Connect("live").CreateStream(7).Publish("test")
	if err := cli.Err(); err != nil {
		panic(err)
	}

	r, err := flv.New("E:\\trailer.flv")
	if err != nil {
		panic(err)
	}
	r.ReadFlvHeader()
	for r.HasMore() {
		t, err := r.ReadFlvTag()
		if err != nil {
			return
		}
		switch t.Header.TagType {
		case 8: //audio
			cli.Audio(t.Header.Timestamp, t.Data.Raw())
		case 9: //video
			cli.Video(t.Header.Timestamp, t.Data.Raw())
		case 18: //meta
			cli.Data(t.Header.Timestamp, t.Data.Raw())
		}
	}
	fmt.Println("推流结束")
}

```