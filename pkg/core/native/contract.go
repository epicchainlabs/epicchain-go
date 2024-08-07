package native

import (
	"strings"

	"github.com/epicchainlabs/epicchain-go/pkg/config"
	"github.com/epicchainlabs/epicchain-go/pkg/core/interop"
	"github.com/epicchainlabs/epicchain-go/pkg/core/interop/interopnames"
	"github.com/epicchainlabs/epicchain-go/pkg/io"
	"github.com/epicchainlabs/epicchain-go/pkg/util"
	"github.com/epicchainlabs/epicchain-go/pkg/vm/emit"
)

// Contracts is a set of registered native contracts.
type Contracts struct {
	Management *Management
	Ledger     *Ledger
	NEO        *NEO
	GAS        *GAS
	Policy     *Policy
	Oracle     *Oracle
	Designate  *Designate
	Notary     *Notary
	Crypto     *Crypto
	Std        *Std
	Contracts  []interop.Contract
	// persistScript is a vm script which executes "onPersist" method of every native contract.
	persistScript []byte
	// postPersistScript is a vm script which executes "postPersist" method of every native contract.
	postPersistScript []byte
}

// ByHash returns a native contract with the specified hash.
func (cs *Contracts) ByHash(h util.Uint160) interop.Contract {
	for _, ctr := range cs.Contracts {
		if ctr.Metadata().Hash.Equals(h) {
			return ctr
		}
	}
	return nil
}

// ByName returns a native contract with the specified name.
func (cs *Contracts) ByName(name string) interop.Contract {
	name = strings.ToLower(name)
	for _, ctr := range cs.Contracts {
		if strings.ToLower(ctr.Metadata().Name) == name {
			return ctr
		}
	}
	return nil
}

// NewContracts returns a new set of native contracts with new GAS, NEO, Policy, Oracle,
// Designate and (optional) Notary contracts.
func NewContracts(cfg config.ProtocolConfiguration) *Contracts {
	cs := new(Contracts)

	mgmt := newManagement()
	cs.Management = mgmt
	cs.Contracts = append(cs.Contracts, mgmt)

	s := newStd()
	cs.Std = s
	cs.Contracts = append(cs.Contracts, s)

	c := newCrypto()
	cs.Crypto = c
	cs.Contracts = append(cs.Contracts, c)

	ledger := newLedger()
	cs.Ledger = ledger
	cs.Contracts = append(cs.Contracts, ledger)

	gas := newGAS(int64(cfg.InitialGASSupply), cfg.P2PSigExtensions)
	neo := newNEO(cfg)
	policy := newPolicy(cfg.P2PSigExtensions)
	neo.GAS = gas
	neo.Policy = policy
	gas.NEO = neo
	gas.Policy = policy
	mgmt.NEO = neo
	mgmt.Policy = policy
	policy.NEO = neo

	cs.GAS = gas
	cs.NEO = neo
	cs.Policy = policy
	cs.Contracts = append(cs.Contracts, neo, gas, policy)

	desig := newDesignate(cfg.Genesis.Roles)
	desig.NEO = neo
	cs.Designate = desig
	cs.Contracts = append(cs.Contracts, desig)

	oracle := newOracle()
	oracle.GAS = gas
	oracle.NEO = neo
	oracle.Desig = desig
	cs.Oracle = oracle
	cs.Contracts = append(cs.Contracts, oracle)

	if cfg.P2PSigExtensions {
		notary := newNotary()
		notary.GAS = gas
		notary.NEO = neo
		notary.Desig = desig
		notary.Policy = policy
		cs.Notary = notary
		cs.Contracts = append(cs.Contracts, notary)
	}

	return cs
}

// GetPersistScript returns a VM script calling "onPersist" syscall for native contracts.
func (cs *Contracts) GetPersistScript() []byte {
	if cs.persistScript != nil {
		return cs.persistScript
	}
	w := io.NewBufBinWriter()
	emit.Syscall(w.BinWriter, interopnames.SystemContractNativeOnPersist)
	cs.persistScript = w.Bytes()
	return cs.persistScript
}

// GetPostPersistScript returns a VM script calling "postPersist" syscall for native contracts.
func (cs *Contracts) GetPostPersistScript() []byte {
	if cs.postPersistScript != nil {
		return cs.postPersistScript
	}
	w := io.NewBufBinWriter()
	emit.Syscall(w.BinWriter, interopnames.SystemContractNativePostPersist)
	cs.postPersistScript = w.Bytes()
	return cs.postPersistScript
}
