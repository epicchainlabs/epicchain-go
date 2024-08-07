# NeoGo P2P signature collection (notary) service

P2P signature (notary) service is a NeoGo node extension that allows several
parties to sign one transaction independently of chain and without going beyond the
chain environment. The on-chain P2P service is aimed to automate, accelerate and
secure the process of signature collection. The service was initially designed as
a solution for
[multisignature transaction forming](https://github.com/neo-project/neo/issues/1573#issue-600384746)
and described in the [proposal](https://github.com/neo-project/neo/issues/1573#issuecomment-704874472).

The original problem definition:
> Several parties want to sign one transaction, it can either be a set of signatures
> for multisignature signer or multiple signers in one transaction. It's assumed
> that all parties can generate the same transaction (with the same hash) without
> any interaction, which is the case for oracle nodes or NeoFS inner ring nodes.
> 
> As some of the services using this mechanism can be quite sensitive to the
> latency of their requests processing, it should be possible to construct a complete
> transaction within the time frame between two consecutive blocks.


## Components and functionality
The service consists of a native contract and a node module. Native contract is
mostly concerned with verification, fees and payment guarantees, while module is
doing the actual work. It uses generic `Conflicts` and `NotValidBefore`
transaction attributes for its purposes as well as an additional special one
(`Notary assisted`).

A new designated role is added, `P2PNotary`. It can have an arbitrary number of
keys associated with it.

To use the service, one should pay some GAS, so below we operate with `FEE` as a unit of cost
for this service. `FEE` is set to be 0.1 GAS.

We'll also use `NKeys` definition as the number of keys that participate in the
process of signature collection. This is the number of keys that could potentially
sign the transaction, for transactions lacking appropriate witnesses that would be
the number of witnesses, for "M out of N" multisignature scripts that's N, for
combination of K standard signature witnesses and L multisignature "M out of N"
witnesses that's K+N*L.

### Transaction attributes

#### Conflicts

This attribute makes the chain accept one transaction of the two conflicting only
and adds an ability to give a priority to any of the two if needed. This
attribute was originally proposed in
[neo-project/neo#1991](https://github.com/neo-project/neo/issues/1991).

The attribute has Uint256 data inside containing the hash of conflicting
transaction. It is allowed to have multiple attributes of this type.

#### NotValidBefore

This attribute makes transaction invalid before certain height. This attribute
was originally proposed in
[neo-project/neo#1992](https://github.com/neo-project/neo/issues/1992).

The attribute has uint32 data inside which is the block height starting from
which the transaction is considered to be valid. It can be seen as the opposite
of `ValidUntilBlock`. Using both allows to have a window of valid block numbers
that this transaction could be accepted into. Transactions with this attribute
are not accepted into mempool before specified block is persisted.

It can be used to create some transactions in advance with a guarantee that they
won't be accepted until the specified block.

#### NotaryAssisted

This attribute holds one byte containing the number of transactions collected
by the service. It could be 0 for fallback transaction or `NKeys` for a normal
transaction that completed its P2P signature collection. Transactions using this
attribute need to pay additional network fee of (`NKeys`+1)×`FEE`. This attribute
could be only be used by transactions signed by the notary native contract.

### Native Notary contract

It exposes several methods to the outside world:

| Method | Parameters | Return value | Description |
| --- | --- | --- | --- |
| `onNEP17Payment` | `from` (uint160) - GAS sender account.<br>`amount` (int) - amount of GAS to deposit.<br>`data` represents array of two parameters: <br>1. `to` (uint160) - account of the deposit owner.<br>2. `till` (int) - deposit lock height. | `bool` | Automatically called after GAS transfer to Notary native contract address and records deposited amount as belonging to `to` address with a lock till `till` chain's height. Can only be invoked from native GAS contract. Must be witnessed by `from`. `to` can be left unspecified (null), with a meaning that `to` is the same address as `from`. `amount` can't be less than 2×`FEE` for the first deposit call for the `to` address. Each successive deposit call must have `till` value equal to or more than the previous successful call (allowing for renewal), if it has additional amount of GAS it adds up to the already deposited value.|
| `lockDepositUntil` | `address` (uint160) - account of the deposit owner.<br>`till` (int) - new height deposit is valid until (can't be less than previous value and can't be less than the height when transaction is accepted to the chain plus one). | `void` | Updates deposit expiration value. Must be witnessed by `address`. |
| `withdraw` | `from` (uint160) - account of the deposit owner.<br>`to` (uint160) - account to transfer GAS to. | `bool` | Sends all deposited GAS for `from` address to `to` address. Must be witnessed by `from`. `to` can be left unspecified (null), with a meaning that `to` is the same address as `from`. It can only be successful if the lock has already expired, attempting to withdraw the deposit before that height fails. Partial withdrawal is not supported. Returns boolean result, `true` for successful calls and `false` for failed ones. |
| `balanceOf` | `addr` (uint160) - account of the deposit owner. | `int` | Returns deposited GAS amount for specified address (integer). |
| `expirationOf` | `addr` (uint160) - account of the deposit owner. | `int` | Returns deposit lock height for specified address (integer). |
| `verify` | `signature` (signature) - notary node signature bytes for verification. | `bool` | This is used to verify transactions with notary contract specified as a signer, it needs one signature in the invocation script and it checks for this signature to be made by one of designated keys, effectively implementing "1 out of N" multisignature contract. |
| `getMaxNotValidBeforeDelta` | | `int` | Returns `MaxNotValidBeforeDelta` constraint. Default value is 140. |
| `setMaxNotValidBeforeDelta` | `value` (int) | `void` | Set `MaxNotValidBeforeDelta` constraint. Must be witnessed by committee. |

See the [Notary deposit guide](#1.-Notary-deposit) section on how to deposit
funds to Notary native contract and manage the deposit.

### P2PNotaryRequest payload

A new broadcasted payload type is introduced for notary requests. It's
distributed via regular inv-getdata mechanism like transactions, blocks or
consensus payloads. An ordinary P2P node verifies it, saves in a structure
similar to mempool and relays. This payload has witness (standard
single-signature contract) attached signing all of the payload.

This payload has two incomplete transactions inside:

- *Fallback tx*. This transaction has P2P Notary contract as a sender and service
  request sender as an additional signer. It can't have a witness for Notary
  contract, but it must have proper witness for request sender. It must have
  `NotValidBefore` attribute that is no more than `MaxNotValidBeforeDelta` higher
  than the current chain height and it must have `Conflicts` attribute with the
  hash of the main transaction. It at the same time must have `Notary assisted`
  attribute with a count of zero.
- *Main tx*. This is the one that actually needs to be completed; it:
  1. *either* doesn't have all witnesses attached
  2. *or* has a partial multisignature only
  3. *or* have not all witnesses attached and some of the rest are partial multisignature
  
  This transaction must have `Notary assisted` attribute with a count of `NKeys`
  (and Notary contract as one of the signers).

See the [Notary request submission guide](#2-request-submission) to learn how to
construct and send the payload.

### Notary node module

Node module with the designated key monitors the network for `P2PNotaryRequest`
payloads. It maintains a list of current requests grouped by main transaction
hash. When it receives enough requests to correctly construct all transaction
witnesses, it does so, adds a witness of its own (for Notary contract witness) and
sends the resulting transaction to the network.

If the main transaction with all witnesses attached still can't be validated
due to any fee (or other) issues, the node waits for `NotValidBefore` block of
the fallback transaction to be persisted.

If `NotValidBefore` block is persisted and there are still some signatures
missing (or the resulting transaction is invalid), the module sends all the
associated fallback transactions for the main transaction.

After processing, service request is deleted from the module.

See the [NeoGo P2P signature extensions](#NeoGo P2P signature extensions) on how
to enable notary-related extensions on chain and
[NeoGo Notary service node module](#NeoGo Notary service node module) on how to
set up Notary service node.

## Environment setup

To run P2P signature collection service on your network, you need to do:
* Set up [`P2PSigExtensions`](#NeoGo P2P signature extensions) for all nodes in
  the network.
* Set notary node keys in `RoleManagement` native contract.
* [Configure](#NeoGo Notary service node module) and run appropriate number of
  notary nodes with keys specified in `RoleManagement` native contract (at least
  one node is necessary to complete signature collection).

After service is running, you can [create and send](#Notary request lifecycle guide)
notary requests to the network.

### NeoGo P2P signature extensions

As far as Notary service is an extension of the standard NeoGo node, it should be
enabled and properly configured before usage.

#### Configuration

To enable P2P signature extensions add `P2PSigExtensions` subsection set to
`true` to `ProtocolConfiguration` section of your node config. This enables all
notary-related logic in the network, i.e. allows your node to accept and validate
`NotValidBefore`, `Conflicts` and `NotaryAssisted` transaction attribute, handle,
verify and broadcast `P2PNotaryRequest` P2P payloads, properly initialize native
Notary contract and designate `P2PNotary` node role in RoleManagement native
contract.

Currently, Notary contract activation height is not configurable and is always
set to 0 (if `P2PSigExtensions` are enabled).

Note, that even if `P2PSigExtensions` config subsection enables notary-related
logic in the network, it still does not turn your node into notary service node.
To enable notary service node functionality refer to the
[NeoGo Notary service](#NeoGo-Notary-service-node-module) documentation.

##### Example

```
  P2PSigExtensions: true
```


### NeoGo Notary service node module

NeoGo node can act as notary service node (the node that accumulates notary
requests, collects signatures and releases fully-signed transactions). It must
have a wallet with a key belonging to one of network's designated notary nodes
(stored in `RoleManagement` native contract). Also, the node must be connected to
a network with enabled P2P signature extensions, otherwise problems with states
and peer disconnections will occur.

Notary service node doesn't need [RPC service](rpc.md) to be enabled because it
receives notary requests and broadcasts completed transactions via P2P protocol.
However, enabling [RPC service](rpc.md) allows to send notary requests directly
to the notary service node and avoid P2P communication delays.

#### Configuration

To enable notary service node check firstly that
[P2PSignatureExtensions](#NeoGo P2P signature extensions) are properly set up.
Then add `P2PNotary` subsection to `ApplicationConfiguration` section of your
node config.

Parameters:
* `Enabled`: boolean value, enables/disables the service node, `true` for service
  node to be enabled
* `UnlockWallet`: notary node wallet configuration:
    - `Path`: path to NEP-6 wallet.
    - `Password`: password for the account to be used by notary node.

##### Example

```
P2PNotary:
  Enabled: true
  UnlockWallet:
    Path: "/notary_node_wallet.json"
    Password: "pass"
```


## Notary request lifecycle guide

Below are presented all stages each P2P signature collection request goes through. Use
stages 1 and 2 to create, sign and submit P2P notary request. Stage 3 is
performed by the notary service; it does not require user's intervention and is given
for informational purposes. Stage 4 contains advice to check for notary request
results.

### 1. Notary deposit

To guarantee that payment to the notary node will still be done if things go wrong,
sender's deposit to the Notary native contract is used. Before the notary request will be
submitted, you need to deposit enough GAS to the contract, otherwise, request
won't pass verification.

Notary native contract supports `onNEP17Payment` method. Thus, to deposit funds to
the Notary native contract, transfer the desired amount of GAS to the contract
address. Use
[func (*Client) TransferNEP17](https://pkg.go.dev/github.com/epicchainlabs/epicchain-go@v0.97.2/pkg/rpcclient#Client.TransferNEP17)
with the `data` parameter matching the following requirements:
- `data` should be an array of two elements: `to` and `till`.
- `to` denotes the receiver of the deposit. It can be nil in case `to` equals
  the GAS sender.
- `till` denotes chain's height before which deposit is locked and can't be
  withdrawn. `till` can't be less than the current chain height. `till`
  can't be less than the current `till` value for the deposit if the deposit
  already exists. `till` can be set to the provided value iff the transaction
  sender is the owner of the deposit, otherwise the provided `till` value will
  be overridden by the system. If the sender is not the deposit owner, the
  overridden `till` value is either set to be the current chain height + 5760
  (for the newly added deposit) or set to the old `till` value (for the existing
  deposit).

Note, that the first deposit call for the `to` address can't transfer less than 2×`FEE` GAS.
Deposit is allowed for renewal, i.e. consequent `deposit` calls for the same `to`
address add up a specified amount to the already deposited value.

After GAS transfer is successfully submitted to the chain, use [Notary native
contract API](#Native Notary contract) to manage your deposit.

Note, that regular operation flow requires the deposited amount of GAS to be
sufficient to pay for *all* fallback transactions that are currently submitted (all
in-flight notary requests). The default deposit sum for one fallback transaction
should be enough to pay the fallback transaction fees which are system fee and
network fee. Fallback network fee includes (`NKeys`+1)×`FEE` = (0+1)×`FEE` = `FEE`
GAS for `NotaryAssisted` attribute usage and regular fee for the fallback size.
If you need to submit several notary requests, ensure that the deposited amount is
enough to pay for all fallbacks. If the deposited amount is not enough to pay the
fallback fees, `Insufficiend funds` error will be returned from the RPC node
after notary request submission.

### 2. Request submission

Once several parties want to sign one transaction, each of them should generate
the transaction, wrap it into `P2PNotaryRequest` payload and send it to the known RPC
server via [`submitnotaryrequest` RPC call](./rpc.md#submitnotaryrequest-call).
Note, that all parties must generate the same main transaction while fallbacks
can differ.

To create a notary request, you can use [NeoGo RPC client](./rpc.md#Client). The
procedure below uses only basic RPC client functions and show all of the notary
request internals. You can use much simpler Actor interface in the notary
subpackage with an example written in Go doc.

1. Prepare a list of signers with scopes for the main transaction (i.e. the
   transaction that signatures are being collected for, that will be `Signers`
   transaction field). Use the following rules to construct the list:
   * First signer is the one who pays the transaction fees.
   * Each signer is either a multisignature or a standard signature or a contract
     signer.
   * Multisignature and signature signers can be combined.
   * Contract signer can be combined with any other signer.

   Include Notary native contract in the list of signers with the following
   constraints:
   * Notary signer hash is the hash of a native Notary contract that can be fetched
     from the notary RPC client subpackage (notary.Hash)
   * A notary signer must have `None` scope.
   * A notary signer shouldn't be placed at the beginning of the signer list
     because Notary contract does not pay main transaction fees. Other positions
     in the signer list are available for a Notary signer.
2. Construct a script for the main transaction (that will be `Script` transaction
   field) and calculate system fee using regular rules (that will be `SystemFee`
   transaction field). Probably, you'll perform one of these actions:
   1. If the script is a contract method call, use `invokefunction` RPC API
      [func (*Client) InvokeFunction](https://pkg.go.dev/github.com/epicchainlabs/epicchain-go@v0.97.2/pkg/rpcclient#Client.InvokeFunction)
      and fetch the script and the gas consumed from the result.
   2. If the script is more complicated than just a contract method call,
      construct the script manually and use `invokescript` RPC API
      [func (*Client) InvokeScript](https://pkg.go.dev/github.com/epicchainlabs/epicchain-go@v0.97.2/pkg/rpcclient#Client.InvokeScript)
      to fetch the gas consumed from the result.
   3. Or just construct the script and set system fee manually.
3. Calculate the height main transaction is valid until (that will be
   `ValidUntilBlock` transaction field). Consider the following rules for `VUB`
   value estimation:
      * `VUB` value must not be lower than the current chain height.
      * The whole notary request (including fallback transaction) is valid until
        the same `VUB` height.
      * `VUB` value must be lower than notary deposit expiration height. This
        condition guarantees that the deposit won't be withdrawn before notary
        service payment.
      * All parties must provide the same `VUB` for the main transaction. 
4. Construct the list of main transaction attributes (that will be `Attributes`
   transaction field). The list must include `NotaryAssisted` attribute with
   `NKeys` equals the overall number of the keys to be collected excluding notary and
   other contract-based witnesses. For m out of n multisignature request
   `NKeys = n`. For multiple standard signature request, signers `NKeys` equals
   the standard signature signers count.
5. Construct a list of accounts (`wallet.Account` structure from the `wallet`
   package) to calculate network fee for the transaction
   using the following rules. This list will be used in the next step.
   - The number and the order of the accounts should match the transaction signers
     constructed at step 1.
   - An account for a contract signer should have `Contract` field with `Deployed` set
     to `true` if the corresponding contract is deployed on chain.
   - An account for a signature or a multisignature signer should have `Contract` field
     with `Deployed` set to `false` and `Script` set to the signer's verification
     script.
   - An account for a notary signer is **just a placeholder** and should have
     `Contract` field with `Deployed` set to `true`. Its `Invocation` witness script
     parameters will be guessed by the `verify` method signature of Notary contract
     during the network fee calculation at the next step.
     
6. Fill in the main transaction `Nonce` field.
7. Construct a list of main transactions witnesses (that will be `Scripts`
   transaction field). Uses standard rules for witnesses of not yet signed
   transaction (it can't be signed at this stage because network fee is missing):
   - A contract-based witness should have `Invocation` script that pushes arguments
     on stack (it may be empty) and empty `Verification` script. If multiple notary
     requests provide different `Invocation` scripts, the first one will be used
     to construct contract-based witness. If non-empty `Invocation` script is
     specified then it will be taken into account during network fee calculation.
     In case of an empty `Invocation` script, its parameters will be guessed from
     the contract's `verify` signature during network fee calculation.
   - A **Notary contract witness** (which is also a contract-based witness) should
     have empty `Verification` script. `Invocation` script should be either empty
     (allowed for main transaction and forbidden for fallback transaction) or of
     the form [opcode.PUSHDATA1, 64, make([]byte, 64)...] (allowed for main
     transaction and required for fallback transaction by the Notary subsystem to
     pass verification), i.e. to be a placeholder for a notary contract signature.
     Both ways are OK for network fee calculation.
   - A standard signature witness must have regular `Verification` script filled
     even if the `Invocation` script is to be collected from other notary
     requests.
     `Invocation` script **should be empty**.
   - A multisignature witness must have regular `Verification` script filled even
     if `Invocation` script is to be collected from other notary requests.
     `Invocation` script **should be empty**.
8. Calculate network fee for the transaction (that will be `NetworkFee`
   transaction field). Use [func (*Client) CalculateNetworkFee](https://pkg.go.dev/github.com/epicchainlabs/epicchain-go@v0.99.2/pkg/rpcclient#Client.CalculateNetworkFee)
   method with the main transaction given to it.
9. Fill in all signatures that can be provded by the client creating request,
   that includes simple-signature accounts and multisignature accounts where
   the client has one of the keys (in which case an invocation script is
   created that pushes just one signature onto the stack).
10. Sign and submit P2P notary request. Use
    [func (*Actor) Notarize](https://pkg.go.dev/github.com/epicchainlabs/epicchain-go/pkg/rpcclient/notary#Actor.Notarize) for it.
    - Use the signed main transaction from step 9 as `mainTx` argument.
    
    `Notarize` will construct and sign a fallback transaction using `Actor`
    configuration (just a simple `RET` script by default), pack both transactions
    into a P2PNotaryRequest and submit it to the RPC node. It returns hashes of
    the main and fallback transactions as well as their `ValidUntilBlock` value.
    If you need more control over fallback transaction use `Actor` options or
    [func (*Actor) SendRequest](https://pkg.go.dev/github.com/epicchainlabs/epicchain-go/pkg/rpcclient/notary#Actor.SendRequest)
    API.

After P2PNotaryRequests are sent, participants should wait for one of their
transactions (main or fallback) to get accepted into one of subsequent blocks.

### 3. Signatures collection and transaction release

A valid P2PNotaryRequest payload is distributed via P2P network using standard
broadcasting mechanisms until it reaches the designated notary nodes that have the
respective node module active. They collect all payloads for the same main
transaction until enough signatures are collected to create proper witnesses for
it. Then, they attach all witnesses required and send this transaction as usual
and monitor subsequent blocks for its inclusion.

All the operations leading to successful transaction creation are independent
of the chain and could easily be done within one block interval. So, if the
first service request is sent at the current height `H`, the main transaction
is highly likely to be a part of `H+1` block.
 
### 4. Results monitoring

Once the P2PNotaryRequest reaches RPC node, it is added to the notary request pool.
Completed or outdated requests are removed from the pool. Use
[NeoGo notification subsystem](./notifications.md) to track request addition and
removal:

- Use RPC `subscribe` method with `notary_request_event` stream name parameter to
  subscribe to `P2PNotaryRequest` payloads that are added or removed from the
  notary request pool.
- Use `sender` or `signer` filters to filter out a notary request with the desired
  request senders or main tx signers.

Use the notification subsystem to track that the main or the fallback transaction
is accepted to the chain:

- Use RPC `subscribe` method with `transaction_added` stream name parameter to
  subscribe to transactions that are accepted to the chain.
- Use `sender` filter with the Notary native contract hash to filter out fallback
  transactions sent by the Notary node. Use `signer` filter with the notary request
  sender address to filter out the fallback transactions sent by the specified
  sender.
- Use `sender` or `signer` filters to filter out the main transaction with the desired
  sender or signers. You can also filter out the main transaction using Notary
  contract `signer` filter.
- Don't rely on `sender` and `signer` filters only, also check that the received
  transaction has `NotaryAssisted` attribute with the expected `NKeys` value.

Use the notification subsystem to track main or fallback transaction execution
results.

Moreover, you can use all regular RPC calls to track main or fallback transaction
invocation: `getrawtransaction`, `getapplicationlog` etc.

## Notary service use-cases

Several use-cases where Notary subsystem can be applied are described below.

### Committee-signed transactions

The signature collection problem occurs every time committee participants need
to submit a transaction with `m out of n` multisignature, i.g.:
- transfer initial supply of NEO and GAS from a committee multisignature account to
  other addresses on new chain start
- tune valuable chain parameters like gas per block, candidate register price,
  minimum contract deployment fee, Oracle request price, native Policy values etc
- invoke non-native contract methods that require committee multisignature witness

Current solution offers off-chain non-P2P signature collection (either manual
or using some additional network connectivity). It has an obvious downside of
reliance on something external to the network. If it's manual, it's slow and
error-prone; if it's automated, it requires additional protocol for all the
parties involved. For the protocol used by oracle nodes, it also means
nodes explicitly exposing to each other.

With the Notary service all signature collection logic is unified and is on chain already.
The only thing that committee participants should perform is to create and submit
a P2P notary request (can be done independently). Once the sufficient number of signatures
is collected by the service, the desired transaction will be applied and pass committee
witness verification.

### NeoFS Inner Ring nodes

Alphabet nodes of the Inner Ring signature collection is a particular case of committee-signed
transactions. Alphabet nodes multisignature is used for various cases, such as:
- main chain and side chain funds synchronization and withdrawal
- bootstrapping new storage nodes to the network
- network map management and epoch update
- containers and extended ACL management
- side chain governance update

Non-notary on-chain solution for Alphabet nodes multisignature forming is
imitated via contracts collecting invocations of their methods signed by standard
signature of each Alphabet node. Once the sufficient number of invocations is
collected, the invocation is performed.

The described solution has several drawbacks:

- it can only be app-specific (meaning that for every use case this logic would
  be duplicated) because we can't create transactions from transactions (thus
  using proper multisignature account is not possible)
- for `m out of n` multisignature we need at least `m` transactions instead of
  one we really wanted to have; but actually we'll create and process `n` of
  them, so this adds substantial overhead to the chain
- some GAS is inevitably wasted because any invocation could either go the easy
  path (just adding a signature to the list) or really invoke the function we
  wanted to (when all signatures are in place), so test invocations don't really
  help and the user needs to add some GAS to all of these transactions

Notary on-chain Alphabet multisignature collection solution
[uses Notary subsystem](https://github.com/nspcc-dev/neofs-node/pull/404) to
successfully solve these problems, e.g. to calculate precisely the amount of GAS to
pay for contract invocation witnessed by Alphabet nodes (see
[nspcc-dev/neofs-node#47](https://github.com/nspcc-dev/neofs-node/issues/47)),
to reduce container creation delay
(see [nspcc-dev/neofs-node#519](https://github.com/nspcc-dev/neofs-node/issues/519))
etc.

### Contract-sponsored (free) transactions

The original problem and solution are described in
[neo-project/neo#2577](https://github.com/neo-project/neo/issues/2577) discussion.
