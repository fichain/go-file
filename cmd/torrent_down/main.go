package main

import (
	"github.com/fichain/go-file/filechain"
	"time"
)

func newSession() *filechain.Session {
	cfg := filechain.DefaultConfig
//Database:                               "~/rain/session.db",
//	DataDir:                                "~/rain/data",
	cfg.Database = "~/filechain/rainDown/session.db"
	cfg.DataDir = "~/filechain/rainDown/data"
	cfg.PEXEnabled = false
	cfg.RPCEnabled = false
	cfg.DataDirIncludesTorrentID = false
	cfg.LibP2pPort = 10000
	cfg.LibP2pHandShake = time.Second * 10
	cfg.LibP2pBootStrap = []string{"/ip4/127.0.0.1/tcp/4001/p2p/QmXbWBfj7LGMeZqktTi4qUk79cZRwoLCgNWyLeBmt8y2je"}
	cfg.LipP2pRandSeed = 100
	cfg.LibP2pUser = "22222222"
	cfg.DHTBootstrapNodes = []string{}

	s, err := filechain.NewSession(cfg)
	if err != nil {
		panic(err)
	}
	return s
}

func main()  {
	s := newSession()
	fileId := "magnet:?xt=urn:btih:4d8bff5cc79f68f07a85fe3f273f2ccd3b637f30&dn=film"
	//filePath := ""
	_, err := s.AddFileId(fileId, &filechain.AddTorrentOptions{StopAfterDownload: false, Stopped: false})
	if err != nil {
		panic(err)
	}

	for  {
		select {

		}
	}
	//s.AddFileId("")
}



