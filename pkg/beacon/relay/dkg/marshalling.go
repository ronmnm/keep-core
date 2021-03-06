package dkg

import (
	"fmt"
	"math/big"

	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"github.com/keep-network/keep-core/pkg/beacon/relay/group"
	"github.com/keep-network/keep-core/pkg/beacon/relay/registry/gen/pb"
)

// Marshal converts ThresholdSigner to byte array.
func (ts *ThresholdSigner) Marshal() ([]byte, error) {
	return (&pb.ThresholdSigner{
		MemberIndex:          uint32(ts.memberIndex),
		GroupPublicKey:       ts.groupPublicKey.Marshal(),
		GroupPrivateKeyShare: ts.groupPrivateKeyShare.String(),
		GroupPublicKeyShares: marshalGroupPublicKeyShares(ts.groupPublicKeyShares),
	}).Marshal()
}

func marshalGroupPublicKeyShares(
	shares map[group.MemberIndex]*bn256.G2,
) map[uint32][]byte {
	marshalled := make(map[uint32][]byte, len(shares))

	for id, share := range shares {
		marshalled[uint32(id)] = share.Marshal()
	}

	return marshalled
}

// Unmarshal converts a byte array back to ThresholdSigner.
func (ts *ThresholdSigner) Unmarshal(bytes []byte) error {
	pbThresholdSigner := pb.ThresholdSigner{}
	if err := pbThresholdSigner.Unmarshal(bytes); err != nil {
		return err
	}

	groupPublicKey := new(bn256.G2)
	_, err := groupPublicKey.Unmarshal(pbThresholdSigner.GroupPublicKey)
	if err != nil {
		return err
	}

	privateKeyShare := new(big.Int)
	privateKeyShare, ok := privateKeyShare.SetString(pbThresholdSigner.GroupPrivateKeyShare, 10)
	if !ok {
		return fmt.Errorf("Error occured while converting a private key share to string")
	}

	groupPublicKeyShares, err := unmarshalGroupPublicKeyShares(
		pbThresholdSigner.GroupPublicKeyShares,
	)
	if err != nil {
		return err
	}

	ts.memberIndex = group.MemberIndex(pbThresholdSigner.MemberIndex)
	ts.groupPublicKey = groupPublicKey
	ts.groupPrivateKeyShare = privateKeyShare
	ts.groupPublicKeyShares = groupPublicKeyShares

	return nil
}

func unmarshalGroupPublicKeyShares(
	shares map[uint32][]byte,
) (map[group.MemberIndex]*bn256.G2, error) {
	var unmarshalled = make(map[group.MemberIndex]*bn256.G2, len(shares))

	for memberID, shareBytes := range shares {
		share := new(bn256.G2)
		_, err := share.Unmarshal(shareBytes)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal share [%v]", err)
		}

		unmarshalled[group.MemberIndex(memberID)] = share
	}

	return unmarshalled, nil
}
