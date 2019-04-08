package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/henrylee2cn/teleport"
	. "github.com/ourcolour/xnettools/simple/entities"
)

//go:generate go build $GOFILE

func main() {
	defer tp.FlushLogger()
	tp.SetLoggerLevel2(tp.NOTICE)

	// graceful
	go tp.GraceSignal()

	// server peer
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:   true,
		ListenPort:  9090,
		PrintDetail: true,
	})
	// srv.SetTLSConfig(tp.GenerateTLSConfigForServer())

	// router
	srv.RouteCallFunc(Hello)
	srv.RouteCall(new(Math))
	srv.RoutePush(new(Alive))
	srv.RouteCall(new(FileTransmit))

	// broadcast per 5s
	go func() {
		for {
			time.Sleep(time.Second * 5)
			srv.RangeSession(func(sess tp.Session) bool {
				sess.Push(
					"/push/status",
					fmt.Sprintf("this is a broadcast, server time: %v", time.Now()),
				)
				return true
			})
		}
	}()

	// listen and serve
	srv.ListenAndServe()
	select {}
}

func Hello(ctx tp.CallCtx, arg *string) (string, *tp.Rerror) {
	ctx.Infof("Hello world! %s\n", arg)
	return "Reply hello", nil
}

// Math handler
type Math struct {
	tp.CallCtx
}

// Add handles addition request
func (m *Math) Add(arg *[]int) (int, *tp.Rerror) {
	// test query parameter
	tp.Debugf("author: %s", m.PeekMeta("author"))
	// add
	var r int
	for _, a := range *arg {
		r += a
	}
	// response
	return r, nil
}

type Alive struct {
	tp.PushCtx
}

func (this *Alive) HeartBeat(arg *time.Time) *tp.Rerror {
	tp.Debugf("%v", *arg)
	return nil
}

type FileTransmit struct {
	tp.CallCtx
}

func (this *FileTransmit) Send(arg *FileTransmitInfo) (int, *tp.Rerror) {
	if nil == arg {
		tp.Panicf("%s\n", "无效的参数")
	}

	//
	saveDirPath := "/Volumes/Data/"
	savePath := filepath.Join(saveDirPath, arg.FileName)

	fileHandler, err := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		tp.Fatalf("%v", err)
	}
	//os.Chmod(savePath, 0755)

	startPos := arg.BufferSize * arg.BatchNo
	leftLength := arg.FileSize - startPos

	tp.Noticef("CC -->		[Bth: %d of %d] [D: %d of %d] [Lft: %d] %s",
		arg.BatchNo, arg.TotalBatches,
		len(arg.Data), arg.FileSize,
		leftLength,
		arg.FileName,
	)

	sentLength, err := fileHandler.WriteAt(arg.Data, startPos)
	if err != nil {
		tp.Fatalf("%v", err)
	}
	fileHandler.Close()

	if err != nil {
		tp.Fatalf("%v", err)
	}

	if sentLength != len(arg.Data) {
		tp.Fatalf("写入大小与收到数据大小不一致: %d <> %d", sentLength, len(arg.Data))
	}

	return sentLength, tp.ToRerror(err)
}
