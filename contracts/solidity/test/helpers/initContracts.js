import { duration } from './increaseTime';
import { bls } from './data';
const KeepToken = artifacts.require('./KeepToken.sol');
const StakingProxy = artifacts.require('./StakingProxy.sol');
const TokenStaking = artifacts.require('./TokenStaking.sol');
const KeepRandomBeaconFrontendProxy = artifacts.require('./KeepRandomBeaconFrontendProxy.sol');
const KeepRandomBeaconFrontendImplV1 = artifacts.require('./KeepRandomBeaconFrontendImplV1.sol');
const KeepRandomBeaconBackend = artifacts.require('./KeepRandomBeaconBackend.sol');

async function getContracts(accounts) {

  let token, stakingProxy, stakingContract,
    frontendImplV1, frontendProxy, frontend,
    backend;

  let minimumStake = 200000,
    groupThreshold = 15,
    groupSize = 20,
    timeoutInitial = 20,
    timeoutSubmission = 100,
    timeoutChallenge = 60,
    timeDKG = 20,
    resultPublicationBlockStep = 3;

  // Initialize Keep token contract
  token = await KeepToken.new();

  // Initialize staking contract under proxy
  stakingProxy = await StakingProxy.new();
  stakingContract = await TokenStaking.new(token.address, stakingProxy.address, duration.days(30));
  await stakingProxy.authorizeContract(stakingContract.address, {from: accounts[0]})

  // Initialize Keep Random Beacon frontend contract
  frontendImplV1 = await KeepRandomBeaconFrontendImplV1.new();
  frontendProxy = await KeepRandomBeaconFrontendProxy.new(frontendImplV1.address);
  frontend = await KeepRandomBeaconFrontendImplV1.at(frontendProxy.address)

  // Initialize Keep Random Beacon backend contract
  backend = await KeepRandomBeaconBackend.new();
  await backend.initialize(
    stakingProxy.address, frontend.address, minimumStake, groupThreshold,
    groupSize, timeoutInitial, timeoutSubmission, timeoutChallenge, timeDKG, resultPublicationBlockStep,
    bls.groupSignature, bls.groupPubKey
  );
  
  await frontend.initialize(1, 1, backend.address);

  // TODO: replace with a secure authorization protocol (addressed in RFC 4).
  await backend.authorizeStakingContract(stakingContract.address);
  await backend.relayEntry(1, bls.groupSignature, bls.groupPubKey, bls.previousEntry, bls.seed);

  return {
    token: token,
    frontend: frontend,
    backend: backend
  };
};

module.exports.getContracts = getContracts;
