package main

import (
	"io"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/henrylee2cn/teleport"
	. "github.com/ourcolour/xnettools/simple/entities"
)

//go:generate go build $GOFILE

func main() {
	defer tp.SetLoggerLevel("ERROR")()

	cli := tp.NewPeer(tp.PeerConfig{})
	defer cli.Close()
	// cli.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})

	cli.RoutePush(new(Push))

	sess, err := cli.Dial(":9090")
	if err != nil {
		tp.Fatalf("%v", err)
	}

	//
	var reply string
	rerr := sess.Call("/hello", "My name is client", &reply).Rerror()
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("Call hello reply: %s", reply)

	//
	var result int
	rerr = sess.Call("/math/add",
		[]int{1, 2, 3, 4, 5},
		&result,
		tp.WithAddMeta("author", "henrylee2cn"),
	).Rerror()
	if rerr != nil {
		tp.Fatalf("%v", rerr)
	}
	tp.Printf("result: %d", result)

	// 打开文件
	filePath := "/Users/cc/Pictures/Photo by Rodion Kutsaev (pVoEPpLw818).jpg"
	fileHandler, e := os.Open(filePath)
	if nil != e {
		tp.Panicf("%s\n", e.Error())
	}
	defer fileHandler.Close()

	fileStat, e := fileHandler.Stat()
	if nil != e {
		tp.Panicf("%s\n", e.Error())
	}

	const BUFFER_SIZE = 1024 * 10
	var buffer []byte = make([]byte, BUFFER_SIZE)

	var batchNo int64
	var totalBatches int64 = int64(math.Ceil(float64(fileStat.Size()) / float64(BUFFER_SIZE)))
	var leftLength int64 = fileStat.Size()
	var totalLength int64 = fileStat.Size()

	for batchNo = 0; batchNo < totalBatches; batchNo++ {
		recvLength, e := fileHandler.Read(buffer)
		if nil != e {
			if io.EOF != e {
				tp.Panicf("%s\n", e.Error())
			}
			break
		}

		fileBytes := buffer[:recvLength]
		f := &FileTransmitInfo{
			FileName: filepath.Base(filePath),
			FileSize: fileStat.Size(),

			BufferSize: BUFFER_SIZE,
			Data:       fileBytes,

			BatchNo:      batchNo,
			TotalBatches: totalBatches,
		}

		// 发送到服务器端
		var sentLength int64
		rerr = sess.Call(
			"/file_transmit/send",
			f,
			&sentLength,
		).Rerror()
		if rerr != nil {
			tp.Fatalf("%v", rerr)
		}

		leftLength -= sentLength

		tp.Printf("CC -->		[Bth: %d of %d] [D: %d of %d] [Lft: %d] %s",
			f.BatchNo, f.TotalBatches,
			len(f.Data), totalLength,
			leftLength,
			f.FileName,
		)
	}

	//for {
	//	rerr = sess.Push("/alive/heart_beat", time.Now())
	//	if rerr != nil {
	//		tp.Fatalf("%v", rerr)
	//	}
	//	tp.Printf("Call hello reply: %s", reply)
	//
	//	time.Sleep(time.Second * 2)
	//}

	tp.Printf("wait for 3s...")
	time.Sleep(time.Second * 3)
}

// Push push handler
type Push struct {
	tp.PushCtx
}

// Push handles '/push/status' message
func (p *Push) Status(arg *string) *tp.Rerror {
	tp.Printf("%s", *arg)
	return nil
}

//
