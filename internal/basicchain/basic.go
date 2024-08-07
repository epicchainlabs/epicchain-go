package basicchain

import (
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"path"
	"path/filepath"
	"testing"

	"github.com/epicchainlabs/epicchain-go/pkg/core/native"
	"github.com/epicchainlabs/epicchain-go/pkg/core/native/nativenames"
	"github.com/epicchainlabs/epicchain-go/pkg/core/native/noderoles"
	"github.com/epicchainlabs/epicchain-go/pkg/encoding/fixedn"
	"github.com/epicchainlabs/epicchain-go/pkg/io"
	"github.com/epicchainlabs/epicchain-go/pkg/neotest"
	"github.com/epicchainlabs/epicchain-go/pkg/rpcclient/nns"
	"github.com/epicchainlabs/epicchain-go/pkg/util"
	"github.com/epicchainlabs/epicchain-go/pkg/vm/opcode"
	"github.com/epicchainlabs/epicchain-go/pkg/vm/stackitem"
	"github.com/epicchainlabs/epicchain-go/pkg/wallet"
	"github.com/stretchr/testify/require"
)

const neoAmount = 99999000

// Various contract IDs that were deployed to basic chain.
const (
	RublesContractID         = int32(1)
	VerifyContractID         = int32(2)
	VerifyWithArgsContractID = int32(3)
	NNSContractID            = int32(4)
	NFSOContractID           = int32(5)
	StorageContractID        = int32(6)
)

const (
	// RublesOldTestvalue is a value initially stored by `testkey` key inside Rubles contract.
	RublesOldTestvalue = "testvalue"
	// RublesNewTestvalue is an updated value of Rubles' storage item with `testkey` key.
	RublesNewTestvalue = "newtestvalue"
)

// InitSimple initializes chain with simple contracts from 'examples'  folder.
// It's not as complicated as chain got after Init and may be used for tests where
// chain with a small amount of data is needed and for historical functionality testing.
// Needs a path to the root directory.
func InitSimple(t *testing.T, rootpath string, e *neotest.Executor) {
	// examplesPrefix is a prefix of the example smart-contracts.
	var examplesPrefix = filepath.Join(rootpath, "examples")

	deployExample := func(t *testing.T, name string) util.Uint160 {
		_, h := newDeployTx(t, e, e.Validator,
			filepath.Join(examplesPrefix, name, name+".go"),
			filepath.Join(examplesPrefix, name, name+".yml"),
			true)
		return h
	}

	// Block #1: deploy storage contract (examples/storage/storage.go).
	storageHash := deployExample(t, "storage")
	storageValidatorInvoker := e.ValidatorInvoker(storageHash)

	// Block #2: put (1, 1) kv pair.
	storageValidatorInvoker.Invoke(t, 1, "put", 1, 1)

	// Block #3: put (2, 2) kv pair.
	storageValidatorInvoker.Invoke(t, 2, "put", 2, 2)

	// Block #4: update (1, 1) -> (1, 2).
	storageValidatorInvoker.Invoke(t, 1, "put", 1, 2)

	// Block #5: deploy runtime contract (examples/runtime/runtime.go).
	_ = deployExample(t, "runtime")
}

// Init pushes some predefined set  of transactions into the given chain, it needs a path to
// the root project directory.
func Init(t *testing.T, rootpath string, e *neotest.Executor) {
	if !e.Chain.GetConfig().P2PSigExtensions {
		t.Fatal("P2PSigExtensions should be enabled to init basic chain")
	}

	var (
		// examplesPrefix is a prefix of the example smart-contracts.
		examplesPrefix = filepath.Join(rootpath, "examples")
		// testDataPrefix is used to retrieve smart contracts that should be deployed to
		// Basic chain.
		testDataPrefix   = filepath.Join(rootpath, "internal", "basicchain", "testdata")
		notaryModulePath = filepath.Join(rootpath, "pkg", "services", "notary")
	)

	gasHash := e.NativeHash(t, nativenames.Gas)
	neoHash := e.NativeHash(t, nativenames.Neo)
	policyHash := e.NativeHash(t, nativenames.Policy)
	notaryHash := e.NativeHash(t, nativenames.Notary)
	designationHash := e.NativeHash(t, nativenames.Designation)
	t.Logf("native GAS hash: %v", gasHash)
	t.Logf("native NEO hash: %v", neoHash)
	t.Logf("native Policy hash: %v", policyHash)
	t.Logf("native Notary hash: %v", notaryHash)
	t.Logf("Block0 hash: %s", e.Chain.GetHeaderHash(0).StringLE())

	acc0 := e.Validator.(neotest.MultiSigner).Single(2) // priv0 index->order and order->index conversion
	priv0ScriptHash := acc0.ScriptHash()
	acc1 := e.Validator.(neotest.MultiSigner).Single(0) // priv1 index->order and order->index conversion
	priv1ScriptHash := acc1.ScriptHash()
	neoValidatorInvoker := e.ValidatorInvoker(neoHash)
	gasValidatorInvoker := e.ValidatorInvoker(gasHash)
	neoPriv0Invoker := e.NewInvoker(neoHash, acc0)
	gasPriv0Invoker := e.NewInvoker(gasHash, acc0)
	designateSuperInvoker := e.NewInvoker(designationHash, e.Validator, e.Committee)

	deployContractFromPriv0 := func(t *testing.T, path, contractName string, configPath string, expectedID int32) (util.Uint256, util.Uint256, util.Uint160) {
		txDeployHash, cH := newDeployTx(t, e, acc0, path, configPath, true)
		b := e.TopBlock(t)
		return b.Hash(), txDeployHash, cH
	}

	e.CheckGASBalance(t, priv0ScriptHash, big.NewInt(5000_0000)) // gas bounty

	// Block #1: move 1000 GAS and neoAmount NEO to priv0.
	txMoveNeo := neoValidatorInvoker.PrepareInvoke(t, "transfer", e.Validator.ScriptHash(), priv0ScriptHash, neoAmount, nil)
	txMoveGas := gasValidatorInvoker.PrepareInvoke(t, "transfer", e.Validator.ScriptHash(), priv0ScriptHash, int64(fixedn.Fixed8FromInt64(1000)), nil)
	b := e.AddNewBlock(t, txMoveNeo, txMoveGas)
	e.CheckHalt(t, txMoveNeo.Hash(), stackitem.Make(true))
	e.CheckHalt(t, txMoveGas.Hash(), stackitem.Make(true))
	t.Logf("Block1 hash: %s", b.Hash().StringLE())
	bw := io.NewBufBinWriter()
	b.EncodeBinary(bw.BinWriter)
	require.NoError(t, bw.Err)
	jsonB, err := b.MarshalJSON()
	require.NoError(t, err)
	t.Logf("Block1 base64: %s", base64.StdEncoding.EncodeToString(bw.Bytes()))
	t.Logf("Block1 JSON: %s", string(jsonB))
	bw.Reset()
	b.Header.EncodeBinary(bw.BinWriter)
	require.NoError(t, bw.Err)
	jsonH, err := b.Header.MarshalJSON()
	require.NoError(t, err)
	t.Logf("Header1 base64: %s", base64.StdEncoding.EncodeToString(bw.Bytes()))
	t.Logf("Header1 JSON: %s", string(jsonH))
	jsonTxMoveNeo, err := txMoveNeo.MarshalJSON()
	require.NoError(t, err)
	t.Logf("txMoveNeo hash: %s", txMoveNeo.Hash().StringLE())
	t.Logf("txMoveNeo JSON: %s", string(jsonTxMoveNeo))
	t.Logf("txMoveNeo base64: %s", base64.StdEncoding.EncodeToString(txMoveNeo.Bytes()))
	t.Logf("txMoveGas hash: %s", txMoveGas.Hash().StringLE())

	e.EnsureGASBalance(t, priv0ScriptHash, func(balance *big.Int) bool { return balance.Cmp(big.NewInt(1000*native.GASFactor)) >= 0 })
	// info for getblockheader rpc tests
	t.Logf("header hash: %s", b.Hash().StringLE())
	buf := io.NewBufBinWriter()
	b.Header.EncodeBinary(buf.BinWriter)
	t.Logf("header: %s", hex.EncodeToString(buf.Bytes()))

	// Block #2: deploy test_contract (Rubles contract).
	cfgPath := filepath.Join(testDataPrefix, "test_contract.yml")
	block2H, txDeployH, cHash := deployContractFromPriv0(t, filepath.Join(testDataPrefix, "test_contract.go"), "Rubl", cfgPath, RublesContractID)
	t.Logf("txDeploy: %s", txDeployH.StringLE())
	t.Logf("Block2 hash: %s", block2H.StringLE())

	// Block #3: invoke `putValue` method on the test_contract.
	rublPriv0Invoker := e.NewInvoker(cHash, acc0)
	txInvH := rublPriv0Invoker.Invoke(t, true, "putValue", "testkey", RublesOldTestvalue)
	t.Logf("txInv: %s", txInvH.StringLE())

	// Block #4: transfer 1000 NEO from priv0 to priv1.
	neoPriv0Invoker.Invoke(t, true, "transfer", priv0ScriptHash, priv1ScriptHash, 1000, nil)

	// Block #5: initialize rubles contract and transfer 1000 rubles from the contract to priv0.
	initTx := rublPriv0Invoker.PrepareInvoke(t, "init")
	transferTx := e.NewUnsignedTx(t, rublPriv0Invoker.Hash, "transfer", cHash, priv0ScriptHash, 1000, nil)
	e.SignTx(t, transferTx, 1500_0000, acc0) // Set system fee manually to avoid verification failure.
	e.AddNewBlock(t, initTx, transferTx)
	e.CheckHalt(t, initTx.Hash(), stackitem.NewBool(true))
	e.CheckHalt(t, transferTx.Hash(), stackitem.Make(true))
	t.Logf("receiveRublesTx: %v", transferTx.Hash().StringLE())

	// Block #6: transfer 123 rubles from priv0 to priv1
	transferTxH := rublPriv0Invoker.Invoke(t, true, "transfer", priv0ScriptHash, priv1ScriptHash, 123, nil)
	t.Logf("sendRublesTx: %v", transferTxH.StringLE())

	// Block #7: push verification contract into the chain.
	verifyPath := filepath.Join(testDataPrefix, "verify", "verification_contract.go")
	verifyCfg := filepath.Join(testDataPrefix, "verify", "verification_contract.yml")
	_, _, _ = deployContractFromPriv0(t, verifyPath, "Verify", verifyCfg, VerifyContractID)

	// Block #8: deposit some GAS to notary contract for priv0.
	transferTxH = gasPriv0Invoker.Invoke(t, true, "transfer", priv0ScriptHash, notaryHash, 10_0000_0000, []any{priv0ScriptHash, int64(e.Chain.BlockHeight() + 1000)})
	t.Logf("notaryDepositTxPriv0: %v", transferTxH.StringLE())

	// Block #9: designate new Notary node.
	ntr, err := wallet.NewWalletFromFile(path.Join(notaryModulePath, "./testdata/notary1.json"))
	require.NoError(t, err)
	require.NoError(t, ntr.Accounts[0].Decrypt("one", ntr.Scrypt))
	designateSuperInvoker.Invoke(t, stackitem.Null{}, "designateAsRole",
		int64(noderoles.P2PNotary), []any{ntr.Accounts[0].PublicKey().Bytes()})
	t.Logf("Designated Notary node: %s", ntr.Accounts[0].PublicKey().StringCompressed())

	// Block #10: push verification contract with arguments into the chain.
	verifyPath = filepath.Join(testDataPrefix, "verify_args", "verification_with_args_contract.go")
	verifyCfg = filepath.Join(testDataPrefix, "verify_args", "verification_with_args_contract.yml")
	_, _, _ = deployContractFromPriv0(t, verifyPath, "VerifyWithArgs", verifyCfg, VerifyWithArgsContractID) // block #10

	// Block #11: push NameService contract into the chain.
	nsPath := filepath.Join(examplesPrefix, "nft-nd-nns")
	nsConfigPath := filepath.Join(nsPath, "nns.yml")
	_, _, nsHash := deployContractFromPriv0(t, nsPath, nsPath, nsConfigPath, NNSContractID) // block #11
	nsCommitteeInvoker := e.CommitteeInvoker(nsHash)
	nsPriv0Invoker := e.NewInvoker(nsHash, acc0)

	// Block #12: transfer funds to committee for further NS record registration.
	gasValidatorInvoker.Invoke(t, true, "transfer",
		e.Validator.ScriptHash(), e.Committee.ScriptHash(), 1000_00000000, nil) // block #12

	// Block #13: add `.com` root to NNS.
	nsCommitteeInvoker.Invoke(t, stackitem.Null{}, "addRoot", "com") // block #13

	// Block #14: register `neo.com` via NNS.
	registerTxH := nsPriv0Invoker.Invoke(t, true, "register",
		"neo.com", priv0ScriptHash) // block #14
	res := e.GetTxExecResult(t, registerTxH)
	require.Equal(t, 1, len(res.Events)) // transfer
	tokenID, err := res.Events[0].Item.Value().([]stackitem.Item)[3].TryBytes()
	require.NoError(t, err)
	t.Logf("NNS token #1 ID (hex): %s", hex.EncodeToString(tokenID))

	// Block #15: set A record type with priv0 owner via NNS.
	nsPriv0Invoker.Invoke(t, stackitem.Null{}, "setRecord", "neo.com", int64(nns.A), "1.2.3.4") // block #15

	// Block #16: invoke `test_contract.go`: put new value with the same key to check `getstate` RPC call
	txPutNewValue := rublPriv0Invoker.PrepareInvoke(t, "putValue", "testkey", RublesNewTestvalue) // tx1
	// Invoke `test_contract.go`: put values to check `findstates` RPC call.
	txPut1 := rublPriv0Invoker.PrepareInvoke(t, "putValue", "aa", "v1")   // tx2
	txPut2 := rublPriv0Invoker.PrepareInvoke(t, "putValue", "aa10", "v2") // tx3
	txPut3 := rublPriv0Invoker.PrepareInvoke(t, "putValue", "aa50", "v3") // tx4
	e.AddNewBlock(t, txPutNewValue, txPut1, txPut2, txPut3)               // block #16
	e.CheckHalt(t, txPutNewValue.Hash(), stackitem.NewBool(true))
	e.CheckHalt(t, txPut1.Hash(), stackitem.NewBool(true))
	e.CheckHalt(t, txPut2.Hash(), stackitem.NewBool(true))
	e.CheckHalt(t, txPut3.Hash(), stackitem.NewBool(true))

	// Block #17: deploy NeoFS Object contract (NEP11-Divisible).
	nfsPath := filepath.Join(examplesPrefix, "nft-d")
	nfsConfigPath := filepath.Join(nfsPath, "nft.yml")
	_, _, nfsHash := deployContractFromPriv0(t, nfsPath, nfsPath, nfsConfigPath, NFSOContractID) // block #17
	nfsPriv0Invoker := e.NewInvoker(nfsHash, acc0)
	nfsPriv1Invoker := e.NewInvoker(nfsHash, acc1)

	// Block #18: mint 1.00 NFSO token by transferring 10 GAS to NFSO contract.
	containerID := util.Uint256{1, 2, 3}
	objectID := util.Uint256{4, 5, 6}
	txGas0toNFSH := gasPriv0Invoker.Invoke(t, true, "transfer",
		priv0ScriptHash, nfsHash, 10_0000_0000, []any{containerID.BytesBE(), objectID.BytesBE()}) // block #18
	res = e.GetTxExecResult(t, txGas0toNFSH)
	require.Equal(t, 2, len(res.Events)) // GAS transfer + NFSO transfer
	tokenID, err = res.Events[1].Item.Value().([]stackitem.Item)[3].TryBytes()
	require.NoError(t, err)
	t.Logf("NFSO token #1 ID (hex): %s", hex.EncodeToString(tokenID))

	// Block #19: transfer 0.25 NFSO from priv0 to priv1.
	nfsPriv0Invoker.Invoke(t, true, "transfer", priv0ScriptHash, priv1ScriptHash, 25, tokenID, nil) // block #19

	// Block #20: transfer 1000 GAS to priv1.
	gasValidatorInvoker.Invoke(t, true, "transfer", e.Validator.ScriptHash(),
		priv1ScriptHash, int64(fixedn.Fixed8FromInt64(1000)), nil) // block #20

	// Block #21: transfer 0.05 NFSO from priv1 back to priv0.
	nfsPriv1Invoker.Invoke(t, true, "transfer", priv1ScriptHash, priv0ScriptHash, 5, tokenID, nil) // block #21

	// Block #22: deploy storage_contract (Storage contract for `traverseiterator` and `terminatesession` RPC calls test).
	storagePath := filepath.Join(testDataPrefix, "storage", "storage_contract.go")
	storageCfg := filepath.Join(testDataPrefix, "storage", "storage_contract.yml")
	_, _, _ = deployContractFromPriv0(t, storagePath, "Storage", storageCfg, StorageContractID)

	// Block #23: add FAULTed transaction to check WSClient waitloops.
	faultedInvoker := e.NewInvoker(cHash, acc0)
	faultedH := faultedInvoker.InvokeScriptCheckFAULT(t, []byte{byte(opcode.ABORT)}, []neotest.Signer{acc0}, "at instruction 0 (ABORT): ABORT")
	t.Logf("FAULTed transaction:\n\thash LE: %s\n\tblock index: %d", faultedH.StringLE(), e.Chain.BlockHeight())

	// Compile contract to test `invokescript` RPC call
	invokePath := filepath.Join(testDataPrefix, "invoke", "invokescript_contract.go")
	invokeCfg := filepath.Join(testDataPrefix, "invoke", "invoke.yml")
	_, _ = newDeployTx(t, e, acc0, invokePath, invokeCfg, false)

	// Prepare some transaction for future submission.
	txSendRaw := neoPriv0Invoker.PrepareInvoke(t, "transfer", priv0ScriptHash, priv1ScriptHash, int64(fixedn.Fixed8FromInt64(1000)), nil)
	bw.Reset()
	txSendRaw.EncodeBinary(bw.BinWriter)
	t.Logf("sendrawtransaction: \n\tbase64: %s\n\tHash LE: %s", base64.StdEncoding.EncodeToString(bw.Bytes()), txSendRaw.Hash().StringLE())

	sr20, err := e.Chain.GetStateModule().GetStateRoot(20)
	require.NoError(t, err)
	t.Logf("Block #20 stateroot LE: %s", sr20.Root.StringLE())
}

func newDeployTx(t *testing.T, e *neotest.Executor, sender neotest.Signer, sourcePath, configPath string, deploy bool) (util.Uint256, util.Uint160) {
	c := neotest.CompileFile(t, sender.ScriptHash(), sourcePath, configPath)
	t.Logf("contract (%s): \n\tHash: %s\n\tAVM: %s", sourcePath, c.Hash.StringLE(), base64.StdEncoding.EncodeToString(c.NEF.Script))
	if deploy {
		return e.DeployContractBy(t, sender, c, nil), c.Hash
	}
	return util.Uint256{}, c.Hash
}
