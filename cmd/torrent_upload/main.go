package main

import (
	"fmt"
	"github.com/fichain/go-file/filechain"
	"github.com/fichain/go-file/internal/magnet"
	"os"
	"path/filepath"
	"time"
)

func newSession() *filechain.Session {
	cfg := filechain.DefaultConfig
	cfg.Database = "~/filechain/rainUpload/session.db"
	cfg.DataDir = "~/filechain/rainUpload/data"
	cfg.PEXEnabled = false
	cfg.RPCEnabled = false
	cfg.DataDirIncludesTorrentID = false
	cfg.LibP2pPort = 20000
	cfg.LibP2pHandShake = time.Second * 10
	cfg.LibP2pBootStrap = []string{"/ip4/127.0.0.1/tcp/4001/p2p/QmXbWBfj7LGMeZqktTi4qUk79cZRwoLCgNWyLeBmt8y2je"}
	cfg.LipP2pRandSeed = 0
	cfg.LibP2pUser = "11111"
	cfg.DHTBootstrapNodes = []string{}

	s, err := filechain.NewSession(cfg)
	if err != nil {
		panic(err)
	}
	return s
}

func main()  {
	currentPath, _ := os.Getwd()
	fmt.Println("currentPath:", currentPath)

	path := filepath.Join(currentPath, "../../testdata/sample_torrent")
	s := newSession()
	t, err := s.CreateFile(path)
	if err != nil {
		panic(err)
	}

	var ret [20]byte
	copy(ret[:], t.InfoHash())

	ma := magnet.Magnet{
		InfoHash: ret,
		Name: t.Name(),
	}
	fmt.Println("ma info:", ma.String())
	for  {
		select {

		}
	}
	//s.AddFileId("")
}



