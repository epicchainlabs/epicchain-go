package wallet

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/nspcc-dev/neo-go/cli/cmdargs"
	"github.com/nspcc-dev/neo-go/cli/flags"
	"github.com/nspcc-dev/neo-go/cli/input"
	"github.com/nspcc-dev/neo-go/cli/options"
	"github.com/nspcc-dev/neo-go/cli/txctx"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/encoding/address"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/neo"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/manifest"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
	"github.com/urfave/cli"
)

const (
	// EnterPasswordPrompt is a prompt used to ask the user for a password.
	EnterPasswordPrompt = "Enter password > "
	// EnterNewPasswordPrompt is a prompt used to ask the user for a password on
	// account creation.
	EnterNewPasswordPrompt = "Enter new password > "
	// EnterOldPasswordPrompt is a prompt used to ask the user for an old password.
	EnterOldPasswordPrompt = "Enter old password > "
	// ConfirmPasswordPrompt is a prompt used to confirm the password.
	ConfirmPasswordPrompt = "Confirm password > "
)

var (
	errNoPath                 = errors.New("wallet path is mandatory and should be passed using (--wallet, -w) flags or via wallet config using --wallet-config flag")
	errConflictingWalletFlags = errors.New("--wallet flag conflicts with --wallet-config flag, please, provide one of them to specify wallet location")
	errPhraseMismatch         = errors.New("the entered pass-phrases do not match. Maybe you have misspelled them")
	errNoStdin                = errors.New("can't read wallet from stdin for this command")
)

var (
	walletPathFlag = cli.StringFlag{
		Name:  "wallet, w",
		Usage: "Path to the wallet file ('-' to read from stdin); conflicts with --wallet-config flag.",
	}
	walletConfigFlag = cli.StringFlag{
		Name:  "wallet-config",
		Usage: "Path to the wallet config file; conflicts with --wallet flag.",
	}
	wifFlag = cli.StringFlag{
		Name:  "wif",
		Usage: "WIF to import",
	}
	decryptFlag = cli.BoolFlag{
		Name:  "decrypt, d",
		Usage: "Decrypt encrypted keys.",
	}
	inFlag = cli.StringFlag{
		Name:  "in",
		Usage: "file with JSON transaction",
	}
	fromAddrFlag = flags.AddressFlag{
		Name:  "from",
		Usage: "Address to send an asset from",
	}
	toAddrFlag = flags.AddressFlag{
		Name:  "to",
		Usage: "Address to send an asset to",
	}
)

// NewCommands returns 'wallet' command.
func NewCommands() []cli.Command {
	claimFlags := []cli.Flag{
		walletPathFlag,
		walletConfigFlag,
		txctx.GasFlag,
		txctx.SysGasFlag,
		txctx.OutFlag,
		txctx.ForceFlag,
		txctx.AwaitFlag,
		flags.AddressFlag{
			Name:  "address, a",
			Usage: "Address to claim GAS for",
		},
	}
	claimFlags = append(claimFlags, options.RPC...)
	signFlags := []cli.Flag{
		walletPathFlag,
		walletConfigFlag,
		txctx.OutFlag,
		txctx.AwaitFlag,
		inFlag,
		flags.AddressFlag{
			Name:  "address, a",
			Usage: "Address to use",
		},
	}
	signFlags = append(signFlags, options.RPC...)
	return []cli.Command{{
		Name:  "wallet",
		Usage: "create, open and manage a Neo wallet",
		Subcommands: []cli.Command{
			{
				Name:      "claim",
				Usage:     "claim GAS",
				UsageText: "neo-go wallet claim -w wallet [--wallet-config path] [-g gas] [-e sysgas] -a address -r endpoint [-s timeout] [--out file] [--force] [--await]",
				Action:    claimGas,
				Flags:     claimFlags,
			},
			{
				Name:      "init",
				Usage:     "create a new wallet",
				UsageText: "neo-go wallet init -w wallet [--wallet-config path] [-a]",
				Action:    createWallet,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					cli.BoolFlag{
						Name:  "account, a",
						Usage: "Create a new account",
					},
				},
			},
			{
				Name:      "change-password",
				Usage:     "change password for accounts",
				UsageText: "neo-go wallet change-password -w wallet -a address",
				Action:    changePassword,
				Flags: []cli.Flag{
					walletPathFlag,
					flags.AddressFlag{
						Name:  "address, a",
						Usage: "address to change password for",
					},
				},
			},
			{
				Name:      "convert",
				Usage:     "convert addresses from existing Neo Legacy NEP6-wallet to Neo N3 format",
				UsageText: "neo-go wallet convert -w legacywallet [--wallet-config path] -o n3wallet",
				Action:    convertWallet,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					cli.StringFlag{
						Name:  "out, o",
						Usage: "where to write converted wallet",
					},
				},
			},
			{
				Name:      "create",
				Usage:     "add an account to the existing wallet",
				UsageText: "neo-go wallet create -w wallet [--wallet-config path]",
				Action:    addAccount,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
				},
			},
			{
				Name:      "dump",
				Usage:     "check and dump an existing Neo wallet",
				UsageText: "neo-go wallet dump -w wallet [--wallet-config path] [-d]",
				Description: `Prints the given wallet (via -w option or via wallet configuration file) in JSON
   format to the standard output. If -d is given, private keys are unencrypted and
   displayed in clear text on the console! Be very careful with this option and
   don't use it unless you know what you're doing.
`,
				Action: dumpWallet,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					decryptFlag,
				},
			},
			{
				Name:      "dump-keys",
				Usage:     "dump public keys for account",
				UsageText: "neo-go wallet dump-keys -w wallet [--wallet-config path] [-a address]",
				Action:    dumpKeys,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					flags.AddressFlag{
						Name:  "address, a",
						Usage: "address to print public keys for",
					},
				},
			},
			{
				Name:      "export",
				Usage:     "export keys for address",
				UsageText: "export -w wallet [--wallet-config path] [--decrypt] [<address>]",
				Description: `Prints the key for the given account to the standard output. It uses NEP-2
   encrypted format by default (the way NEP-6 wallets store it) or WIF format if
   -d option is given. In the latter case the key can be displayed in clear text
   on the console, so be extremely careful with this option and don't use unless
   you really need it and know what you're doing.
`,
				Action: exportKeys,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					decryptFlag,
				},
			},
			{
				Name:      "import",
				Usage:     "import WIF of a standard signature contract",
				UsageText: "import -w wallet [--wallet-config path] --wif <wif> [--name <account_name>]",
				Action:    importWallet,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					wifFlag,
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Optional account name",
					},
					cli.StringFlag{
						Name:  "contract",
						Usage: "Verification script for custom contracts",
					},
				},
			},
			{
				Name:  "import-multisig",
				Usage: "import multisig contract",
				UsageText: "import-multisig -w wallet [--wallet-config path] [--wif <wif>] [--name <account_name>] --min <m>" +
					" [<pubkey1> [<pubkey2> [...]]]",
				Description: `Imports a standard multisignature contract with "m out of n" signatures required where "m" is
       specified by --min flag and "n" is the length of provided set of public keys. If
       --wif flag is provided, it's used to create an account with the given name (or
       without a name if --name flag is not provided). Otherwise, the command tries to
       find an account with one of the given public keys and convert it to multisig. If
       no suitable account is found and no --wif flag is specified, an error is returned.
`,
				Action: importMultisig,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					wifFlag,
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Optional account name",
					},
					cli.IntFlag{
						Name:  "min, m",
						Usage: "Minimal number of signatures",
					},
				},
			},
			{
				Name:      "import-deployed",
				Usage:     "import deployed contract",
				UsageText: "import-deployed -w wallet [--wallet-config path] --wif <wif> --contract <hash> [--name <account_name>]",
				Action:    importDeployed,
				Flags: append([]cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					wifFlag,
					cli.StringFlag{
						Name:  "name, n",
						Usage: "Optional account name",
					},
					flags.AddressFlag{
						Name:  "contract, c",
						Usage: "Contract hash or address",
					},
				}, options.RPC...),
			},
			{
				Name:      "remove",
				Usage:     "remove an account from the wallet",
				UsageText: "remove -w wallet [--wallet-config path] [--force] --address <addr>",
				Action:    removeAccount,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					txctx.ForceFlag,
					flags.AddressFlag{
						Name:  "address, a",
						Usage: "Account address or hash in LE form to be removed",
					},
				},
			},
			{
				Name:      "sign",
				Usage:     "cosign transaction with multisig/contract/additional account",
				UsageText: "sign -w wallet [--wallet-config path] --address <address> --in <file.in> [--out <file.out>] [-r <endpoint>] [--await]",
				Description: `Signs the given (in file.in) context (which must be a transaction
   signing context) for the given address using the given wallet. This command can
   output the resulting JSON (with additional signature added) right to the console
   (if no file.out and no RPC endpoint specified) or into a file (which can be the
   same as input one). If an RPC endpoint is given it'll also try to construct a
   complete transaction and send it via RPC (printing its hash if everything is OK). 
   If the --await (with a given RPC endpoint) flag is included, the command waits 
   for the transaction to be included in a block before exiting.
`,
				Action: signStoredTransaction,
				Flags:  signFlags,
			},
			{
				Name:      "strip-keys",
				Usage:     "remove private keys for all accounts",
				UsageText: "neo-go wallet strip-keys -w wallet [--wallet-config path] [--force]",
				Description: `Removes private keys for all accounts from the given wallet. Notice,
   this is a very dangerous action (you can lose keys if you don't have a wallet
   backup) that should not be performed unless you know what you're doing. It's
   mostly useful for creation of special wallets that can be used to create
   transactions, but can't be used to sign them (offline signing).
`,
				Action: stripKeys,
				Flags: []cli.Flag{
					walletPathFlag,
					walletConfigFlag,
					txctx.ForceFlag,
				},
			},
			{
				Name:        "nep17",
				Usage:       "work with NEP-17 contracts",
				Subcommands: newNEP17Commands(),
			},
			{
				Name:        "nep11",
				Usage:       "work with NEP-11 contracts",
				Subcommands: newNEP11Commands(),
			},
			{
				Name:        "candidate",
				Usage:       "work with candidates",
				Subcommands: newValidatorCommands(),
			},
		},
	}}
}

func claimGas(ctx *cli.Context) error {
	return handleNeoAction(ctx, func(contract *neo.Contract, shash util.Uint160, _ *wallet.Account) (*transaction.Transaction, error) {
		return contract.TransferUnsigned(shash, shash, big.NewInt(0), nil)
	})
}

func changePassword(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, _, err := openWallet(ctx, false)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()
	if len(wall.Accounts) == 0 {
		return cli.NewExitError("wallet has no accounts", 1)
	}
	addrFlag := ctx.Generic("address").(*flags.Address)
	if addrFlag.IsSet {
		// Check for account presence first before asking for password.
		acc := wall.GetAccount(addrFlag.Uint160())
		if acc == nil {
			return cli.NewExitError("account is missing", 1)
		}
	}

	oldPass, err := input.ReadPassword(EnterOldPasswordPrompt)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("Error reading old password: %w", err), 1)
	}

	for i := range wall.Accounts {
		if addrFlag.IsSet && wall.Accounts[i].Address != addrFlag.String() {
			continue
		}
		err := wall.Accounts[i].Decrypt(oldPass, wall.Scrypt)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("unable to decrypt account %s: %w", wall.Accounts[i].Address, err), 1)
		}
	}

	pass, err := readNewPassword()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("Error reading new password: %w", err), 1)
	}
	for i := range wall.Accounts {
		if addrFlag.IsSet && wall.Accounts[i].Address != addrFlag.String() {
			continue
		}
		err := wall.Accounts[i].Encrypt(pass, wall.Scrypt)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
	}
	err = wall.Save()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("Error saving the wallet: %w", err), 1)
	}
	return nil
}

func convertWallet(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, pass, err := newWalletV2FromFile(ctx.String("wallet"), ctx.String("wallet-config"))
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	out := ctx.String("out")
	if len(out) == 0 {
		return cli.NewExitError("missing out path", 1)
	}
	newWallet, err := wallet.NewWallet(out)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	newWallet.Scrypt = wall.Scrypt

	for _, acc := range wall.Accounts {
		if len(wall.Accounts) != 1 || pass == nil {
			password, err := input.ReadPassword(fmt.Sprintf("Enter password for account %s (label '%s') > ", acc.Address, acc.Label))
			if err != nil {
				return cli.NewExitError(fmt.Errorf("Error reading password: %w", err), 1)
			}
			pass = &password
		}

		newAcc, err := acc.convert(*pass, wall.Scrypt)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
		newWallet.AddAccount(newAcc)
	}
	if err := newWallet.Save(); err != nil {
		return cli.NewExitError(err, 1)
	}
	return nil
}

func addAccount(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, pass, err := openWallet(ctx, true)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()

	if err := createAccount(wall, pass); err != nil {
		return cli.NewExitError(err, 1)
	}

	return nil
}

func exportKeys(ctx *cli.Context) error {
	wall, pass, err := readWallet(ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()

	var addr string

	decrypt := ctx.Bool("decrypt")
	if ctx.NArg() == 0 && decrypt {
		return cli.NewExitError(errors.New("address must be provided if '--decrypt' flag is used"), 1)
	} else if ctx.NArg() > 0 {
		// check address format just to catch possible typos
		addr = ctx.Args().First()
		_, err := address.StringToUint160(addr)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("can't parse address: %w", err), 1)
		}
	}

	var wifs []string

loop:
	for _, a := range wall.Accounts {
		if addr != "" && a.Address != addr {
			continue
		}

		for i := range wifs {
			if a.EncryptedWIF == wifs[i] {
				continue loop
			}
		}

		wifs = append(wifs, a.EncryptedWIF)
	}

	for _, wif := range wifs {
		if decrypt {
			if pass == nil {
				password, err := input.ReadPassword(EnterPasswordPrompt)
				if err != nil {
					return cli.NewExitError(fmt.Errorf("Error reading password: %w", err), 1)
				}
				pass = &password
			}

			pk, err := keys.NEP2Decrypt(wif, *pass, wall.Scrypt)
			if err != nil {
				return cli.NewExitError(err, 1)
			}

			wif = pk.WIF()
		}

		fmt.Fprintln(ctx.App.Writer, wif)
	}

	return nil
}

func importMultisig(ctx *cli.Context) error {
	var (
		label  *string
		acc    *wallet.Account
		accPub *keys.PublicKey
	)

	wall, pass, err := openWallet(ctx, true)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()

	m := ctx.Int("min")
	if ctx.NArg() < m {
		return cli.NewExitError(errors.New("insufficient number of public keys"), 1)
	}

	args := []string(ctx.Args())
	pubs := make([]*keys.PublicKey, len(args))

	for i := range args {
		pubs[i], err = keys.NewPublicKeyFromString(args[i])
		if err != nil {
			return cli.NewExitError(fmt.Errorf("can't decode public key %d: %w", i, err), 1)
		}
	}

	if ctx.IsSet("name") {
		l := ctx.String("name")
		label = &l
	}

loop:
	for _, pub := range pubs {
		for _, wallAcc := range wall.Accounts {
			if wallAcc.ScriptHash().Equals(pub.GetScriptHash()) {
				if acc != nil {
					// Multiple matching accounts found, fallback to WIF-based conversion.
					acc = nil
					break loop
				}
				acc = new(wallet.Account)
				*acc = *wallAcc
				accPub = pub
			}
		}
	}

	if acc != nil {
		err = acc.ConvertMultisigEncrypted(accPub, m, pubs)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
		if label != nil {
			acc.Label = *label
		}
		if err := addAccountAndSave(wall, acc); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	}

	if !ctx.IsSet("wif") {
		return cli.NewExitError(errors.New("none of the provided public keys correspond to an existing key in the wallet or multiple matching accounts found in the wallet, and no WIF is provided"), 1)
	}
	acc, err = newAccountFromWIF(ctx.App.Writer, ctx.String("wif"), wall.Scrypt, label, pass)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	if err := acc.ConvertMultisig(m, pubs); err != nil {
		return cli.NewExitError(err, 1)
	}

	if err := addAccountAndSave(wall, acc); err != nil {
		return cli.NewExitError(err, 1)
	}

	return nil
}

func importDeployed(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, pass, err := openWallet(ctx, true)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()

	rawHash := ctx.Generic("contract").(*flags.Address)
	if !rawHash.IsSet {
		return cli.NewExitError("contract hash was not provided", 1)
	}

	var label *string
	if ctx.IsSet("name") {
		l := ctx.String("name")
		label = &l
	}
	acc, err := newAccountFromWIF(ctx.App.Writer, ctx.String("wif"), wall.Scrypt, label, pass)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()

	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	cs, err := c.GetContractStateByHash(rawHash.Uint160())
	if err != nil {
		return cli.NewExitError(fmt.Errorf("can't fetch contract info: %w", err), 1)
	}
	md := cs.Manifest.ABI.GetMethod(manifest.MethodVerify, -1)
	if md == nil || md.ReturnType != smartcontract.BoolType {
		return cli.NewExitError("contract has no `verify` method with boolean return", 1)
	}
	acc.Address = address.Uint160ToString(cs.Hash)
	acc.Contract.Script = cs.NEF.Script
	acc.Contract.Parameters = acc.Contract.Parameters[:0]
	for _, p := range md.Parameters {
		acc.Contract.Parameters = append(acc.Contract.Parameters, wallet.ContractParam{
			Name: p.Name,
			Type: p.Type,
		})
	}
	acc.Contract.Deployed = true

	if err := addAccountAndSave(wall, acc); err != nil {
		return cli.NewExitError(err, 1)
	}

	return nil
}

func importWallet(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, pass, err := openWallet(ctx, true)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()

	var label *string
	if ctx.IsSet("name") {
		l := ctx.String("name")
		label = &l
	}

	acc, err := newAccountFromWIF(ctx.App.Writer, ctx.String("wif"), wall.Scrypt, label, pass)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	if ctrFlag := ctx.String("contract"); ctrFlag != "" {
		ctr, err := hex.DecodeString(ctrFlag)
		if err != nil {
			return cli.NewExitError("invalid contract", 1)
		}
		acc.Contract.Script = ctr
	}

	if err := addAccountAndSave(wall, acc); err != nil {
		return cli.NewExitError(err, 1)
	}

	return nil
}

func removeAccount(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, _, err := openWallet(ctx, true)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()

	addr := ctx.Generic("address").(*flags.Address)
	if !addr.IsSet {
		return cli.NewExitError("valid account address must be provided", 1)
	}
	acc := wall.GetAccount(addr.Uint160())
	if acc == nil {
		return cli.NewExitError("account wasn't found", 1)
	}

	if !ctx.Bool("force") {
		fmt.Fprintf(ctx.App.Writer, "Account %s will be removed. This action is irreversible.\n", addr.Uint160())
		if ok := askForConsent(ctx.App.Writer); !ok {
			return nil
		}
	}

	if err := wall.RemoveAccount(acc.Address); err != nil {
		return cli.NewExitError(fmt.Errorf("error on remove: %w", err), 1)
	}
	if err := wall.Save(); err != nil {
		return cli.NewExitError(fmt.Errorf("error while saving wallet: %w", err), 1)
	}
	return nil
}

func askForConsent(w io.Writer) bool {
	response, err := input.ReadLine("Are you sure? [y/N]: ")
	if err == nil {
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		}
	}
	fmt.Fprintln(w, "Cancelled.")
	return false
}

func dumpWallet(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, pass, err := readWallet(ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()
	if ctx.Bool("decrypt") {
		if pass == nil {
			password, err := input.ReadPassword(EnterPasswordPrompt)
			if err != nil {
				return cli.NewExitError(fmt.Errorf("Error reading password: %w", err), 1)
			}
			pass = &password
		}
		for i := range wall.Accounts {
			// Just testing the decryption here.
			err := wall.Accounts[i].Decrypt(*pass, wall.Scrypt)
			if err != nil {
				return cli.NewExitError(err, 1)
			}
		}
	}
	fmtPrintWallet(ctx.App.Writer, wall)
	return nil
}

func dumpKeys(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, _, err := readWallet(ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()
	accounts := wall.Accounts

	addrFlag := ctx.Generic("address").(*flags.Address)
	if addrFlag.IsSet {
		acc := wall.GetAccount(addrFlag.Uint160())
		if acc == nil {
			return cli.NewExitError("account is missing", 1)
		}
		accounts = []*wallet.Account{acc}
	}

	hasPrinted := false
	for _, acc := range accounts {
		pub, ok := vm.ParseSignatureContract(acc.Contract.Script)
		if ok {
			if hasPrinted {
				fmt.Fprintln(ctx.App.Writer)
			}
			fmt.Fprintf(ctx.App.Writer, "%s (simple signature contract):\n", acc.Address)
			fmt.Fprintln(ctx.App.Writer, hex.EncodeToString(pub))
			hasPrinted = true
			continue
		}
		n, bs, ok := vm.ParseMultiSigContract(acc.Contract.Script)
		if ok {
			if hasPrinted {
				fmt.Fprintln(ctx.App.Writer)
			}
			fmt.Fprintf(ctx.App.Writer, "%s (%d out of %d multisig contract):\n", acc.Address, n, len(bs))
			for i := range bs {
				fmt.Fprintln(ctx.App.Writer, hex.EncodeToString(bs[i]))
			}
			hasPrinted = true
			continue
		}
		if addrFlag.IsSet {
			return cli.NewExitError(fmt.Errorf("unknown script type for address %s", address.Uint160ToString(addrFlag.Uint160())), 1)
		}
	}
	return nil
}

func stripKeys(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	wall, _, err := readWallet(ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer wall.Close()
	if !ctx.Bool("force") {
		fmt.Fprintln(ctx.App.Writer, "All private keys for all accounts will be removed from the wallet. This action is irreversible.")
		if ok := askForConsent(ctx.App.Writer); !ok {
			return nil
		}
	}
	for _, a := range wall.Accounts {
		a.EncryptedWIF = ""
	}
	if err := wall.Save(); err != nil {
		return cli.NewExitError(fmt.Errorf("error while saving wallet: %w", err), 1)
	}
	return nil
}

func createWallet(ctx *cli.Context) error {
	if err := cmdargs.EnsureNone(ctx); err != nil {
		return err
	}
	path := ctx.String("wallet")
	configPath := ctx.String("wallet-config")

	if len(path) != 0 && len(configPath) != 0 {
		return errConflictingWalletFlags
	}
	if len(path) == 0 && len(configPath) == 0 {
		return cli.NewExitError(errNoPath, 1)
	}
	var pass *string
	if len(configPath) != 0 {
		cfg, err := options.ReadWalletConfig(configPath)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
		path = cfg.Path
		pass = &cfg.Password
	}
	wall, err := wallet.NewWallet(path)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	if err := wall.Save(); err != nil {
		return cli.NewExitError(err, 1)
	}

	if ctx.Bool("account") {
		if err := createAccount(wall, pass); err != nil {
			return cli.NewExitError(err, 1)
		}
		defer wall.Close()
	}

	fmtPrintWallet(ctx.App.Writer, wall)
	fmt.Fprintf(ctx.App.Writer, "wallet successfully created, file location is %s\n", wall.Path())
	return nil
}

func readAccountInfo() (string, string, error) {
	name, err := readAccountName()
	if err != nil {
		return "", "", err
	}
	phrase, err := readNewPassword()
	if err != nil {
		return "", "", err
	}
	return name, phrase, nil
}

func readAccountName() (string, error) {
	return input.ReadLine("Enter the name of the account > ")
}

func readNewPassword() (string, error) {
	phrase, err := input.ReadPassword(EnterNewPasswordPrompt)
	if err != nil {
		return "", fmt.Errorf("Error reading password: %w", err)
	}
	phraseCheck, err := input.ReadPassword(ConfirmPasswordPrompt)
	if err != nil {
		return "", fmt.Errorf("Error reading password: %w", err)
	}

	if phrase != phraseCheck {
		return "", errPhraseMismatch
	}
	return phrase, nil
}

func createAccount(wall *wallet.Wallet, pass *string) error {
	var (
		name, phrase string
		err          error
	)
	if pass == nil {
		name, phrase, err = readAccountInfo()
		if err != nil {
			return err
		}
	} else {
		phrase = *pass
	}
	return wall.CreateAccount(name, phrase)
}

func openWallet(ctx *cli.Context, canUseWalletConfig bool) (*wallet.Wallet, *string, error) {
	path, pass, err := getWalletPathAndPass(ctx, canUseWalletConfig)
	if err != nil {
		return nil, nil, cli.NewExitError(fmt.Errorf("failed to get wallet path or password: %w", err), 1)
	}
	if path == "-" {
		return nil, nil, errNoStdin
	}
	w, err := wallet.NewWalletFromFile(path)
	if err != nil {
		return nil, nil, cli.NewExitError(fmt.Errorf("failed to read wallet: %w", err), 1)
	}
	return w, pass, nil
}

func readWallet(ctx *cli.Context) (*wallet.Wallet, *string, error) {
	path, pass, err := getWalletPathAndPass(ctx, true)
	if err != nil {
		return nil, nil, err
	}
	if path == "-" {
		w := &wallet.Wallet{}
		if err := json.NewDecoder(os.Stdin).Decode(w); err != nil {
			return nil, nil, fmt.Errorf("js %w", err)
		}
		return w, nil, nil
	}
	w, err := wallet.NewWalletFromFile(path)
	if err != nil {
		return nil, nil, err
	}
	return w, pass, nil
}

// getWalletPathAndPass retrieves wallet path from context or from wallet configuration file.
// If wallet configuration file is specified, then account password is returned.
func getWalletPathAndPass(ctx *cli.Context, canUseWalletConfig bool) (string, *string, error) {
	path, configPath := ctx.String("wallet"), ctx.String("wallet-config")
	if !canUseWalletConfig && len(configPath) != 0 {
		return "", nil, errors.New("can't use wallet configuration file for this command")
	}
	if len(path) != 0 && len(configPath) != 0 {
		return "", nil, errConflictingWalletFlags
	}
	if len(path) == 0 && len(configPath) == 0 {
		return "", nil, errNoPath
	}
	var pass *string
	if len(configPath) != 0 {
		cfg, err := options.ReadWalletConfig(configPath)
		if err != nil {
			return "", nil, err
		}
		path = cfg.Path
		pass = &cfg.Password
	}
	return path, pass, nil
}

func newAccountFromWIF(w io.Writer, wif string, scrypt keys.ScryptParams, label *string, pass *string) (*wallet.Account, error) {
	var (
		phrase, name string
		err          error
	)
	if pass != nil {
		phrase = *pass
	}
	if label == nil {
		name, err = readAccountName()
		if err != nil {
			return nil, fmt.Errorf("failed to read account label: %w", err)
		}
	} else {
		name = *label
	}
	// note: NEP2 strings always have length of 58 even though
	// base58 strings can have different lengths even if slice lengths are equal
	if len(wif) == 58 {
		if pass == nil {
			phrase, err = input.ReadPassword(EnterPasswordPrompt)
			if err != nil {
				return nil, fmt.Errorf("error reading password: %w", err)
			}
		}

		acc, err := wallet.NewAccountFromEncryptedWIF(wif, phrase, scrypt)
		if err != nil {
			// If password from wallet config wasn't OK then retry with the user input,
			// see the https://github.com/nspcc-dev/neo-go/issues/2883#issuecomment-1399923088.
			if pass == nil {
				return nil, err
			}
			phrase, err = input.ReadPassword(EnterPasswordPrompt)
			if err != nil {
				return nil, fmt.Errorf("error reading password: %w", err)
			}
			acc, err = wallet.NewAccountFromEncryptedWIF(wif, phrase, scrypt)
			if err != nil {
				return nil, err
			}
		}
		acc.Label = name
		return acc, nil
	}

	acc, err := wallet.NewAccountFromWIF(wif)
	if err != nil {
		return nil, err
	}

	fmt.Fprintln(w, "Provided WIF was unencrypted. Wallet can contain only encrypted keys.")
	if pass == nil {
		phrase, err = readNewPassword()
		if err != nil {
			return nil, fmt.Errorf("failed to read new password: %w", err)
		}
	}

	acc.Label = name
	if err := acc.Encrypt(phrase, scrypt); err != nil {
		return nil, err
	}

	return acc, nil
}

func addAccountAndSave(w *wallet.Wallet, acc *wallet.Account) error {
	for i := range w.Accounts {
		if w.Accounts[i].Address == acc.Address {
			return fmt.Errorf("address '%s' is already in wallet", acc.Address)
		}
	}

	w.AddAccount(acc)
	return w.Save()
}

func fmtPrintWallet(w io.Writer, wall *wallet.Wallet) {
	b, _ := wall.JSON()
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, string(b))
	fmt.Fprintln(w, "")
}
