package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"time"

	//crypto "github.com/libp2p/go-libp2p-crypto"
	dht "github.com/libp2p/go-libp2p-kad-dht"

	ma "github.com/multiformats/go-multiaddr"
)

func main()  {
	ctx := context.Background()

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	sourceMultiAddr, _ := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/4001")

	//r := mrand.New(mrand.NewSource(int64(10000000)))
	//
	//prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	//
	//if err != nil {
	//	panic(err)
	//}
	//by, err := prvKey.Raw()
	//if err != nil {
	//	panic(err)
	//}

	pkS := "308204a40201000282010100c3899c92ee1009f21a22c068d7c41704c4871045bea31b4cb2ce6f2584803690260501e4143cf24da27eaec118417a67d584ed9404ff4ed6f6d67a3b23b2815344f9b433a53c6eadf0f94a688d439a28fa21295d4b9611ede6b3e4d84ec2d34c3c68583a5d73669a36be3a8495bc9115f6100942007db6d98423afc61ccf9cad673d73e204631d820a756aa6350960792b2dcd510da0e06de9bfef59c69042e600d1c8ed9c5825d8e6c05e0cf45fd73df474067e48eddf8fd3f92f020990978196a7b54cd4f28fecdbef5fbe82029e78f08d5a0b5c08393f85cc84b35071031f422eff3d8e3616eb9e212aa784683caaa7f34cbdd1eaa2eabcababb7567bc46502030100010282010100bd7d60046b3b93c7d058190a00fe802818a3a2bb53f110859549c4202175766adecd3f75ddbeea391ddd925081c7026e1957063cc952f8fe0c9af03cdb6d2332a4c72f40554269279b3c9a4513908d96643f3aacb4912bb2d63d42e9f3f98d76759bd0d44eb78498b1b04b592d1a5da7609b4dbd6e686588092be42d22c276db491c899cf0cf0657281ce22a32d360d05e1d2f9f7723a463e46d347d4f2b84390d57599e4a7fbe40187bb58e8114c564714d036a6d14c92886d1c9d37242fac032a476d3b096265e4e5365171fd2e8e8802f3a72e0a05309cd578b21644dcb6028a54ab1a4ae03b294c03ea26e21dcafab509664c8a484236e6ffdf063df790102818100cf026a810918af9b72f3b745f95286797f02cfa2baa324aabca258563d43bc358c89302a29b4c70227655953d37b9d93fe2319e8df9cb9559c2473505889554d1b0ab570a03aa5760e16b05e7c3217d9fe70b84f9f85e42162a57285df26f2924b721e23edb358f0072444805dd341545022b6c258c7038b03fc4eefceba418502818100f1d02cf3028be2c764210e3e4806b6620fe6bbbfe33a1e1e074f4b77206dc03d57b822860c9bdebb0d713232cf3ecbfab2914d5bdb543c6a82059188cd56e362e79afe8574c2ea9129c263810e53dd4f9b1e6f5f1001f15e98e9afe8ceabe93bdad0709010750921ba5e11c7bec51d72c4c6a7ae2a483e6a70f2314d0eaf7d6102818030b5b3d3eb0d08fd3dfe4508cd12f31b919c5ab942cb72ac4e38b12a91bde7827e3025ca360818afd40f50069e83bedf7cf44b7b756a8e5daba114153ad00de757ce9c45051ee7a230cc7bf1afada5d920baeb53a908bb5673bcd486d5ac77759f151a2c80192c7b4662ed4f7b446361f07d4a9dce7ffdc06f4ea6505d478c7502818100ca343c0211350c33938518b5f7e0b50e1721e809b366dbdc5c5c704c732f933b3868df659c91929473cf1e1ce2b42e39baac4a35ef97e6d561586ab42c90e59fc4f014b96043c58611975d4183cb991a8229d71374fd4aeac18f57eba33699d7d547cb788c6a717264b758c2e0c14fb8b2d7334c2e4b2ef62ef0374daa6410c1028180756b6755a7727d02e41cc44f0fb314dd5dca8f7efbf90834ab9df04568f58032561196eba191058d78c04b24ce866ec59350974c9b6dae082750e9e0db3149970573010cbe4bf5292abcf0efb0b721a4ad8d5366e0ad16c74790eff94e1fadf76cccce5c35a3a48677492b0d3dfcb0d765c469395481df76770e65abe5e015f8"
	pkB, err := hex.DecodeString(pkS)
	if err != nil {
		panic(err)
	}
	//hex.DecodeString()

	//crypto.UnmarshalRsaPrivateKey()
	pk, err := crypto.UnmarshalRsaPrivateKey(pkB)
	if err != nil {
		panic(err)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(sourceMultiAddr),
		//libp2p.NATPortMap(),
		//libp2p.DefaultSecurity,
		//libp2p.EnableRelay(circuit.OptActive, circuit.OptHop, circuit.OptDiscovery),
		libp2p.Identity(pk),
	}

	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		panic(err)
	}

	fmt.Println("This node: ", host.ID().Pretty(), " ", host.Addrs())

	//db, err := leveldb.NewDatastore("/Users/rain/filechain/leveldb", nil)

	d, err := dht.New(ctx, host, dht.Mode(dht.ModeServer), dht.MaxRecordAge(time.Second * 10))
	//dht.Datastore(db)
	if err != nil {
		panic(err)
	}

	d.RoutingTable().Print()

	//if err = kademliaDHT.Bootstrap(ctx); err != nil {
	//	panic(err)
	//}

	t := time.NewTicker(3 * time.Second)
	for {
		<-t.C
		d.RoutingTable().Print()
		//<- d.ForceRefresh()
	}
	//select {}
}
