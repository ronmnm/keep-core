import exceptThrow from './helpers/expectThrow';
import mineBlocks from './helpers/mineBlocks';
import generateTickets from './helpers/generateTickets';
import stakeDelegate from './helpers/stakeDelegate';
import {getContracts} from './helpers/initContracts';


contract('TestKeepGroupSelection', function(accounts) {

  let token, backend,
  owner = accounts[0], magpie = accounts[1], signature, delegation,
  operator1 = accounts[2], tickets1,
  operator2 = accounts[3], tickets2,
  operator3 = accounts[4], tickets3;

  beforeEach(async () => {

    let contracts = await getContracts(accounts);
    token = contracts.token;
    backend = contracts.backend;
    let stakingContract = await backend.stakingContract();
    let minimumStake = await backend.minimumStake();

    await stakeDelegate(stakingContract, token, owner, operator1, magpie, minimumStake.mul(web3.utils.toBN(2000)))
    await stakeDelegate(stakingContract, token, owner, operator2, magpie, minimumStake.mul(web3.utils.toBN(2000)))
    await stakeDelegate(stakingContract, token, owner, operator3, magpie, minimumStake.mul(web3.utils.toBN(3000)))

    tickets1 = generateTickets(await backend.groupSelectionSeed(), operator1, 2000);
    tickets2 = generateTickets(await backend.groupSelectionSeed(), operator2, 2000);
    tickets3 = generateTickets(await backend.groupSelectionSeed(), operator3, 3000);

  });

  it("should be able to get staking weight", async function() {
    assert.isTrue(web3.utils.toBN(2000).eq(await backend.stakingWeight(operator1)), "Should have expected staking weight.");
    assert.isTrue(web3.utils.toBN(3000).eq(await backend.stakingWeight(operator3)), "Should have expected staking weight.");
  });

  it("should fail to get selected tickets before challenge period is over", async function() {
    await exceptThrow(backend.selectedTickets());
  });

  it("should fail to get selected participants before challenge period is over", async function() {
    await exceptThrow(backend.selectedParticipants());
  });

  it("should be able to get selected tickets and participants after challenge period is over", async function() {

    let groupSize = await backend.groupSize();

    for (let i = 0; i < groupSize*2; i++) {
      await backend.submitTicket(tickets1[i].value, operator1, tickets1[i].virtualStakerIndex, {from: operator1});
    }

    mineBlocks(await backend.ticketChallengeTimeout());
    let selectedTickets = await backend.selectedTickets();
    assert.equal(selectedTickets.length, groupSize, "Should be trimmed to groupSize length.");

    let selectedParticipants = await backend.selectedParticipants();
    assert.equal(selectedParticipants.length, groupSize, "Should be trimmed to groupSize length.");
  });

  it("should be able to output submited tickets in ascending ordered", async function() {

    let tickets = [];

    await backend.submitTicket(tickets1[0].value, operator1, tickets1[0].virtualStakerIndex, {from: operator1});
    tickets.push(tickets1[0].value);

    await backend.submitTicket(tickets2[0].value, operator2, tickets2[0].virtualStakerIndex, {from: operator2});
    tickets.push(tickets2[0].value);

    await backend.submitTicket(tickets3[0].value, operator3, tickets3[0].virtualStakerIndex, {from: operator3});
    tickets.push(tickets3[0].value);

    tickets = tickets.sort(function(a, b){return a-b}); // Sort numbers in ascending order

    // Test tickets ordering
    let orderedTickets = await backend.orderedTickets();
    assert.isTrue(orderedTickets[0].eq(tickets[0]), "Tickets should be in ascending order.");
    assert.isTrue(orderedTickets[1].eq(tickets[1]), "Tickets should be in ascending order.");
    assert.isTrue(orderedTickets[2].eq(tickets[2]), "Tickets should be in ascending order.");

  });

  it("should be able to submit a ticket during ticket submission period", async function() {
    await backend.submitTicket(tickets1[0].value, operator1, tickets1[0].virtualStakerIndex, {from: operator1});
    let proof = await backend.getTicketProof(tickets1[0].value);
    assert.isTrue(proof[1].eq(web3.utils.toBN(operator1)), "Should be able to get submitted ticket proof.");
    assert.equal(proof[2], tickets1[0].virtualStakerIndex, "Should be able to get submitted ticket proof.");
  });

  it("should be able to verify a ticket", async function() {

    await backend.submitTicket(tickets1[0].value, operator1, 1, {from: operator1});

    assert.isTrue(await backend.cheapCheck(
      operator1, operator1, 1
    ), "Should be able to verify a valid ticket.");
    
    assert.isTrue(await backend.costlyCheck(
      operator1, tickets1[0].value, operator1, tickets1[0].virtualStakerIndex
    ), "Should be able to verify a valid ticket.");
  
    assert.isFalse(await backend.costlyCheck(
      operator1, 0, operator1, tickets1[0].virtualStakerIndex
    ), "Should fail verifying invalid ticket.");

  });
});
