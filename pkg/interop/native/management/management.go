/*
Package management provides an interface to ContractManagement native contract.
It allows to get/deploy/update contracts as well as get/set deployment fee.
*/
package management

import (
	"github.com/nspcc-dev/neo-go/pkg/interop"
	"github.com/nspcc-dev/neo-go/pkg/interop/contract"
	"github.com/nspcc-dev/neo-go/pkg/interop/iterator"
	"github.com/nspcc-dev/neo-go/pkg/interop/neogointernal"
)

// Hash represents Management contract hash.
const Hash = "\xfd\xa3\xfa\x43\x46\xea\x53\x2a\x25\x8f\xc4\x97\xdd\xad\xdb\x64\x37\xc9\xfd\xff"

// IDHash is an ID/Hash pair returned by the iterator from the GetContractHashes method.
type IDHash struct {
	// ID is a 32-bit number, but it's represented in big endian form
	// natively, because that's the key scheme used by ContractManagement.
	ID   []byte
	Hash interop.Hash160
}

// Deploy represents `deploy` method of Management native contract.
func Deploy(script, manifest []byte) *Contract {
	return neogointernal.CallWithToken(Hash, "deploy",
		int(contract.All), script, manifest).(*Contract)
}

// DeployWithData represents `deploy` method of Management native contract.
func DeployWithData(script, manifest []byte, data any) *Contract {
	return neogointernal.CallWithToken(Hash, "deploy",
		int(contract.All), script, manifest, data).(*Contract)
}

// Destroy represents `destroy` method of Management native contract.
func Destroy() {
	neogointernal.CallWithTokenNoRet(Hash, "destroy", int(contract.States|contract.AllowNotify))
}

// GetContract represents `getContract` method of Management native contract.
func GetContract(addr interop.Hash160) *Contract {
	return neogointernal.CallWithToken(Hash, "getContract", int(contract.ReadStates), addr).(*Contract)
}

// GetContractByID represents `getContractById` method of the Management native contract.
func GetContractByID(id int) *Contract {
	return neogointernal.CallWithToken(Hash, "getContractById", int(contract.ReadStates), id).(*Contract)
}

// GetContractHashes represents `getContractHashes` method of the Management
// native contract. It returns an Iterator over the list of non-native contract
// hashes. Each iterator value can be cast to IDHash. Use [iterator] interop
// package to work with the returned Iterator.
func GetContractHashes() iterator.Iterator {
	return neogointernal.CallWithToken(Hash, "getContractHashes", int(contract.ReadStates)).(iterator.Iterator)
}

// GetMinimumDeploymentFee represents `getMinimumDeploymentFee` method of Management native contract.
func GetMinimumDeploymentFee() int {
	return neogointernal.CallWithToken(Hash, "getMinimumDeploymentFee", int(contract.ReadStates)).(int)
}

// HasMethod represents `hasMethod` method of Management native contract. It allows to check
// if the "hash" contract has a method named "method" with parameters number equal to "pcount".
func HasMethod(hash interop.Hash160, method string, pcount int) bool {
	return neogointernal.CallWithToken(Hash, "hasMethod", int(contract.ReadStates), hash, method, pcount).(bool)
}

// SetMinimumDeploymentFee represents `setMinimumDeploymentFee` method of Management native contract.
func SetMinimumDeploymentFee(value int) {
	neogointernal.CallWithTokenNoRet(Hash, "setMinimumDeploymentFee", int(contract.States), value)
}

// Update represents `update` method of Management native contract.
func Update(script, manifest []byte) {
	neogointernal.CallWithTokenNoRet(Hash, "update",
		int(contract.All), script, manifest)
}

// UpdateWithData represents `update` method of Management native contract.
func UpdateWithData(script, manifest []byte, data any) {
	neogointernal.CallWithTokenNoRet(Hash, "update",
		int(contract.All), script, manifest, data)
}
