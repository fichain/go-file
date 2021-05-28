package filechain

import (
	"crypto/rand"
	"github.com/fichain/go-file/external/resumer/boltdbresumer"
	"github.com/libp2p/go-libp2p-core/crypto"
)

func (s *Session)getCurrentUserKey(spec *boltdbresumer.SessionSpec) (crypto.PrivKey, error) {
	if spec.UserPrivk != nil && len(spec.UserPrivk) != 0 {
		pk, err := crypto.UnmarshalRsaPrivateKey(spec.UserPrivk)
		if err != nil {
			s.log.Errorln("Unmarshal RsaPrivateKey error! re generate!", err)
		} else {
			s.log.Debugln("get private key from db success!")
			return pk, nil
		}
	}

	s.log.Infoln("no p2p private key, start generate!")

	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		s.log.Errorln("generate privatekey error!!!", err)
		return nil,err
	}

	privB, err := priv.Raw()
	if err != nil {
		s.log.Errorln("generate privatekey error!!!", err)
		return nil, err
	}
	spec.UserPrivk = privB
	s.sessionResumer.Write(spec)
	return priv, nil
}
