package ethereum

import (
	"fmt"
	"math/big"
	"os"
	"time"

	relaychain "github.com/keep-network/keep-core/pkg/beacon/relay/chain"
	relayconfig "github.com/keep-network/keep-core/pkg/beacon/relay/config"
	"github.com/keep-network/keep-core/pkg/beacon/relay/event"
	"github.com/keep-network/keep-core/pkg/gen/async"
)

// ThresholdRelay converts from ethereumChain to beacon.ChainInterface.
func (ec *ethereumChain) ThresholdRelay() relaychain.Interface {
	return ec
}

func (ec *ethereumChain) GetConfig() (relayconfig.Chain, error) {
	size, err := ec.keepGroupContract.GroupSize()
	if err != nil {
		return relayconfig.Chain{}, fmt.Errorf("error calling GroupSize: [%v]", err)
	}

	threshold, err := ec.keepGroupContract.GroupThreshold()
	if err != nil {
		return relayconfig.Chain{}, fmt.Errorf("error calling GroupThreshold: [%v]", err)
	}

	return relayconfig.Chain{
		GroupSize: size,
		Threshold: threshold,
	}, nil
}

func (ec *ethereumChain) SubmitGroupPublicKey(
	groupID string,
	key [96]byte,
) *async.GroupRegistrationPromise {
	groupRegistrationPromise := &async.GroupRegistrationPromise{}

	err := ec.keepRandomBeaconContract.WatchSubmitGroupPublicKeyEvent(
		func(
			groupPublicKey []byte,
			requestID *big.Int,
			activationBlockHeight *big.Int,
		) {
			err := groupRegistrationPromise.Fulfill(&event.GroupRegistration{
				GroupPublicKey:        groupPublicKey,
				RequestID:             requestID,
				ActivationBlockHeight: activationBlockHeight,
			})
			if err != nil {
				fmt.Fprintf(
					os.Stderr,
					"fulfilling promise failed with: [%v].\n",
					err,
				)
			}
		},
		func(err error) error {
			return groupRegistrationPromise.Fail(
				fmt.Errorf(
					"entry of group key failed with: [%v]",
					err,
				),
			)
		},
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"watch group public key event failed with: [%v].\n",
			err,
		)
		return groupRegistrationPromise
	}

	_, err = ec.keepRandomBeaconContract.SubmitGroupPublicKey(key[:], big.NewInt(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to submit GroupPublicKey [%v].\n", err)
		return groupRegistrationPromise
	}

	return groupRegistrationPromise
}

func (ec *ethereumChain) SubmitRelayEntry(
	newEntry *event.Entry,
) *async.RelayEntryPromise {
	relayEntryPromise := &async.RelayEntryPromise{}

	err := ec.keepRandomBeaconContract.WatchRelayEntryGenerated(
		func(
			requestID *big.Int,
			requestResponse *big.Int,
			requestGroupID *big.Int,
			previousEntry *big.Int,
			blockNumber *big.Int,
		) {
			var value [32]byte
			copy(value[:], requestResponse.Bytes()[:32])

			err := relayEntryPromise.Fulfill(&event.Entry{
				RequestID:     requestID,
				Value:         value,
				GroupID:       requestGroupID,
				PreviousEntry: previousEntry,
				Timestamp:     time.Now().UTC(),
			})
			if err != nil {
				fmt.Fprintf(
					os.Stderr,
					"execution of fulfilling promise failed with: [%v]",
					err,
				)
			}
		},
		func(err error) error {
			return relayEntryPromise.Fail(
				fmt.Errorf(
					"entry of relay submission failed with: [%v]",
					err,
				),
			)
		},
	)
	if err != nil {
		promiseErr := relayEntryPromise.Fail(
			fmt.Errorf(
				"watch relay entry failed with: [%v]",
				err,
			),
		)
		if promiseErr != nil {
			fmt.Fprintf(
				os.Stderr,
				"execution of failing promise failed with: [%v]",
				promiseErr,
			)
		}
		return relayEntryPromise
	}

	groupSignature := big.NewInt(int64(0)).SetBytes(newEntry.Value[:])
	_, err = ec.keepRandomBeaconContract.SubmitRelayEntry(
		newEntry.RequestID,
		newEntry.GroupID,
		newEntry.PreviousEntry,
		groupSignature,
	)
	if err != nil {
		promiseErr := relayEntryPromise.Fail(
			fmt.Errorf(
				"submitting relay entry to chain failed with: [%v]",
				err,
			),
		)
		if promiseErr != nil {
			fmt.Printf(
				"execution of failing promise failed with: [%v]",
				promiseErr,
			)
		}
		return relayEntryPromise
	}

	return relayEntryPromise
}

func (ec *ethereumChain) OnRelayEntryGenerated(handle func(entry *event.Entry)) {
	err := ec.keepRandomBeaconContract.WatchRelayEntryGenerated(
		func(
			requestID *big.Int,
			requestResponse *big.Int,
			requestGroupID *big.Int,
			previousEntry *big.Int,
			blockNumber *big.Int,
		) {
			var value [32]byte
			copy(value[:], requestResponse.Bytes()[:32])

			handle(&event.Entry{
				RequestID:     requestID,
				Value:         value,
				GroupID:       requestGroupID,
				PreviousEntry: previousEntry,
				Timestamp:     time.Now().UTC(),
			})
		},
		func(err error) error {
			return fmt.Errorf(
				"watch relay entry failed with: [%v]",
				err,
			)
		},
	)
	if err != nil {
		fmt.Printf(
			"watch relay entry failed with: [%v]",
			err,
		)
	}
}

// OnRelayEntryRequested registers a callback function for a new
// relay request on chain.
func (ec *ethereumChain) OnRelayEntryRequested(
	handle func(request *event.Request),
) {
	err := ec.keepRandomBeaconContract.WatchRelayEntryRequested(
		func(
			requestID *big.Int,
			payment *big.Int,
			blockReward *big.Int,
			seed *big.Int,
			blockNumber *big.Int,
		) {
			handle(&event.Request{
				RequestID:   requestID,
				Payment:     payment,
				BlockReward: blockReward,
				Seed:        seed,
			})
		},
		func(err error) error {
			return fmt.Errorf("relay request event failed with %v", err)
		},
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"watch relay request failed with: [%v]",
			err,
		)
	}
}

// AddStaker is a temporary function for Milestone 1 that adds a
// staker to the group contract.
func (ec *ethereumChain) AddStaker(
	groupMemberID string,
) *async.StakerRegistrationPromise {
	onStakerAddedPromise := &async.StakerRegistrationPromise{}

	if len(groupMemberID) != 32 {
		err := onStakerAddedPromise.Fail(
			fmt.Errorf(
				"groupMemberID wrong length, need 32, got %d",
				len(groupMemberID),
			),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Promise Fail failed [%v].\n", err)
		}
		return onStakerAddedPromise
	}

	err := ec.keepGroupContract.WatchOnStakerAdded(
		func(index int, groupMemberID []byte) {
			err := onStakerAddedPromise.Fulfill(&event.StakerRegistration{
				Index:         index,
				GroupMemberID: string(groupMemberID),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Promise Fulfill failed [%v].\n", err)
			}
		},
		func(err error) error {
			return onStakerAddedPromise.Fail(
				fmt.Errorf(
					"adding new staker failed with: [%v]",
					err,
				),
			)
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to watch OnStakerAdded [%v].\n", err)
		return onStakerAddedPromise
	}

	index, err := ec.keepGroupContract.GetNStaker()
	if err != nil {
		fmt.Printf("Error: failed on call to GetNStaker: [%v]\n", err)
		return onStakerAddedPromise
	}

	tx, err := ec.keepGroupContract.AddStaker(index, groupMemberID)
	if err != nil {
		fmt.Printf(
			"on staker added failed with: [%v]",
			err,
		)
	}

	ec.tx = tx

	return onStakerAddedPromise
}

// GetStakerList is a temporary function for Milestone 1 that
// gets back the list of stakers.
func (ec *ethereumChain) GetStakerList() ([]string, error) {
	max, err := ec.keepGroupContract.GetNStaker()
	if err != nil {
		err = fmt.Errorf("failed on call to GetNStaker: [%v]", err)
		return []string{}, err
	}

	if max == 0 {
		return []string{}, nil
	}

	listOfStakers := make([]string, 0, max)
	for ii := 0; ii < max; ii++ {
		aStaker, err := ec.keepGroupContract.GetStaker(ii)
		if err != nil {
			return []string{},
				fmt.Errorf("at postion %d out of %d error: [%v]", ii, max, err)
		}
		listOfStakers = append(listOfStakers, string(aStaker))
	}

	return listOfStakers, nil
}

func (ec *ethereumChain) OnGroupRegistered(
	handle func(key *event.GroupRegistration),
) {
	err := ec.keepRandomBeaconContract.WatchSubmitGroupPublicKeyEvent(
		func(
			groupPublicKey []byte,
			requestID *big.Int,
			activationBlockHeight *big.Int,
		) {
			handle(&event.GroupRegistration{
				GroupPublicKey:        groupPublicKey,
				RequestID:             requestID,
				ActivationBlockHeight: activationBlockHeight,
			})
		},
		func(err error) error {
			return fmt.Errorf("entry of group key failed with: [%v]", err)
		},
	)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"watch group public key event failed with: [%v].\n",
			err,
		)
	}
}
