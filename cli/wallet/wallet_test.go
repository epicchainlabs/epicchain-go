package wallet_test

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chzyer/readline"
	"github.com/epicchainlabs/epicchain-go/internal/testcli"
	"github.com/epicchainlabs/epicchain-go/pkg/config"
	"github.com/epicchainlabs/epicchain-go/pkg/crypto/hash"
	"github.com/epicchainlabs/epicchain-go/pkg/crypto/keys"
	"github.com/epicchainlabs/epicchain-go/pkg/encoding/address"
	"github.com/epicchainlabs/epicchain-go/pkg/smartcontract"
	"github.com/epicchainlabs/epicchain-go/pkg/util"
	"github.com/epicchainlabs/epicchain-go/pkg/wallet"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWalletAccountRemove(t *testing.T) {
	tmpDir := t.TempDir()
	e := testcli.NewExecutor(t, false)

	walletPath := filepath.Join(tmpDir, "wallet.json")
	e.In.WriteString("acc1\r")
	e.In.WriteString("pass\r")
	e.In.WriteString("pass\r")
	e.Run(t, "neo-go", "wallet", "init", "--wallet", walletPath, "--account")

	e.In.WriteString("acc2\r")
	e.In.WriteString("pass\r")
	e.In.WriteString("pass\r")
	e.Run(t, "neo-go", "wallet", "create", "--wallet", walletPath)

	w, err := wallet.NewWalletFromFile(walletPath)
	require.NoError(t, err)

	t.Run("missing wallet", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "remove")
	})
	t.Run("missing address", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "remove", "--wallet", walletPath)
	})
	t.Run("invalid address", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "remove", "--wallet", walletPath,
			"--address", util.Uint160{}.StringLE())
	})

	addr := w.Accounts[0].Address
	t.Run("askForConsent > no", func(t *testing.T) {
		e.In.WriteString("no")
		e.Run(t, "neo-go", "wallet", "remove", "--wallet", walletPath,
			"--address", addr)
		actual, err := wallet.NewWalletFromFile(walletPath)
		require.NoError(t, err)
		require.Equal(t, w, actual)
	})

	e.Run(t, "neo-go", "wallet", "remove", "--wallet", walletPath,
		"--address", addr, "--force")

	actual, err := wallet.NewWalletFromFile(walletPath)
	require.NoError(t, err)
	require.Equal(t, 1, len(actual.Accounts))
	require.Equal(t, w.Accounts[1], actual.Accounts[0])
}

func TestWalletChangePassword(t *testing.T) {
	tmpDir := t.TempDir()
	e := testcli.NewExecutor(t, false)

	walletPath := filepath.Join(tmpDir, "wallet.json")
	e.In.WriteString("acc1\r")
	e.In.WriteString("pass\r")
	e.In.WriteString("pass\r")
	e.Run(t, "neo-go", "wallet", "init", "--wallet", walletPath, "--account")

	e.In.WriteString("acc2\r")
	e.In.WriteString("pass\r")
	e.In.WriteString("pass\r")
	e.Run(t, "neo-go", "wallet", "create", "--wallet", walletPath)

	w, err := wallet.NewWalletFromFile(walletPath)
	require.NoError(t, err)

	addr1 := w.Accounts[0].Address
	addr2 := w.Accounts[1].Address
	t.Run("missing wallet path", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "change-password")
	})
	t.Run("EOF reading old password", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "change-password", "--wallet", walletPath)
	})
	t.Run("bad old password", func(t *testing.T) {
		e.In.WriteString("ssap\r")
		e.In.WriteString("aaa\r") // Pretend for the password to be fine.
		e.In.WriteString("aaa\r")

		e.RunWithError(t, "neo-go", "wallet", "change-password", "--wallet", walletPath)
	})
	t.Run("no account", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "change-password", "--wallet", walletPath, "--address", "NVTiAjNgagDkTr5HTzDmQP9kPwPHN5BgVq")
	})
	t.Run("bad new password, multiaccount", func(t *testing.T) {
		e.In.WriteString("pass\r")
		e.In.WriteString("pass1\r")
		e.In.WriteString("pass2\r")
		e.RunWithError(t, "neo-go", "wallet", "change-password", "--wallet", walletPath)
	})
	t.Run("good, multiaccount", func(t *testing.T) {
		e.In.WriteString("pass\r")
		e.In.WriteString("asdf\r")
		e.In.WriteString("asdf\r")
		e.Run(t, "neo-go", "wallet", "change-password", "--wallet", walletPath)
	})
	t.Run("good, single account", func(t *testing.T) {
		e.In.WriteString("asdf\r")
		e.In.WriteString("jkl\r")
		e.In.WriteString("jkl\r")
		e.Run(t, "neo-go", "wallet", "change-password", "--wallet", walletPath, "--address", addr1)
	})
	t.Run("bad, different passwords", func(t *testing.T) {
		e.In.WriteString("jkl\r")
		e.RunWithError(t, "neo-go", "wallet", "change-password", "--wallet", walletPath)
	})
	t.Run("good, second account", func(t *testing.T) {
		e.In.WriteString("asdf\r")
		e.In.WriteString("jkl\r")
		e.In.WriteString("jkl\r")
		e.Run(t, "neo-go", "wallet", "change-password", "--wallet", walletPath, "--address", addr2)
	})
	t.Run("good, second multiaccount", func(t *testing.T) {
		e.In.WriteString("jkl\r")
		e.In.WriteString("pass\r")
		e.In.WriteString("pass\r")
		e.Run(t, "neo-go", "wallet", "change-password", "--wallet", walletPath)
	})
}

func TestWalletInit(t *testing.T) {
	e := testcli.NewExecutor(t, false)

	t.Run("missing path", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "init")
	})
	t.Run("invalid path", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "init", "--wallet", t.TempDir())
	})
	t.Run("good: no account", func(t *testing.T) {
		walletPath := filepath.Join(t.TempDir(), "wallet.json")
		e.Run(t, "neo-go", "wallet", "init", "--wallet", walletPath)
		w, err := wallet.NewWalletFromFile(walletPath)
		require.NoError(t, err)
		require.Equal(t, 0, len(w.Accounts))
	})
	t.Run("with account", func(t *testing.T) {
		walletPath := filepath.Join(t.TempDir(), "wallet.json")
		t.Run("missing acc name", func(t *testing.T) {
			e.RunWithError(t, "neo-go", "wallet", "init", "--wallet", walletPath, "--account")
		})
		t.Run("missing pass", func(t *testing.T) {
			e.In.WriteString("acc\r")
			e.RunWithError(t, "neo-go", "wallet", "init", "--wallet", walletPath, "--account")
		})
		t.Run("missing second pass", func(t *testing.T) {
			e.In.WriteString("acc\r")
			e.In.WriteString("pass\r")
			e.RunWithError(t, "neo-go", "wallet", "init", "--wallet", walletPath, "--account")
		})
		e.In.WriteString("acc\r")
		e.In.WriteString("pass\r")
		e.In.WriteString("pass\r")
		e.Run(t, "neo-go", "wallet", "init", "--wallet", walletPath, "--account")
		w, err := wallet.NewWalletFromFile(walletPath)
		require.NoError(t, err)
		require.Equal(t, 1, len(w.Accounts))
		require.Equal(t, "acc", w.Accounts[0].Label)
	})
	t.Run("with wallet config", func(t *testing.T) {
		tmp := t.TempDir()
		walletPath := filepath.Join(tmp, "wallet.json")
		configPath := filepath.Join(tmp, "config.yaml")
		cfg := config.Wallet{
			Path:     walletPath,
			Password: "pass",
		}
		res, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, res, 0666))
		e.Run(t, "neo-go", "wallet", "init", "--wallet-config", configPath, "--account")
		w, err := wallet.NewWalletFromFile(walletPath)
		require.NoError(t, err)
		require.Equal(t, 1, len(w.Accounts))
		require.Equal(t, "", w.Accounts[0].Label)
	})

	tmpDir := t.TempDir()
	walletPath := filepath.Join(tmpDir, "wallet.json")
	e.Run(t, "neo-go", "wallet", "init", "--wallet", walletPath)

	t.Run("terminal escape codes", func(t *testing.T) {
		walletPath := filepath.Join(tmpDir, "walletrussian.json")
		bksp := string([]byte{
			byte(readline.CharBackward),
			byte(readline.CharDelete),
		})
		e.In.WriteString("буквыы" +
			bksp + bksp + bksp +
			"andmore\r")
		e.In.WriteString("пароу" + bksp + "ль\r")
		e.In.WriteString("пароль\r")
		e.Run(t, "neo-go", "wallet", "init", "--account",
			"--wallet", walletPath)

		w, err := wallet.NewWalletFromFile(walletPath)
		require.NoError(t, err)
		require.Len(t, w.Accounts, 1)
		require.Equal(t, "букandmore", w.Accounts[0].Label)
		require.NoError(t, w.Accounts[0].Decrypt("пароль", w.Scrypt))
	})

	t.Run("CreateAccount", func(t *testing.T) {
		t.Run("stdin", func(t *testing.T) {
			e.RunWithError(t, "neo-go", "wallet", "create", "--wallet", "-")
		})
		t.Run("passwords mismatch", func(t *testing.T) {
			e.In.WriteString("testname\r")
			e.In.WriteString("testpass\r")
			e.In.WriteString("badpass\r")
			e.RunWithError(t, "neo-go", "wallet", "create", "--wallet", walletPath)
		})
		e.In.WriteString("testname\r")
		e.In.WriteString("testpass\r")
		e.In.WriteString("testpass\r")
		e.Run(t, "neo-go", "wallet", "create", "--wallet", walletPath)

		w, err := wallet.NewWalletFromFile(walletPath)
		require.NoError(t, err)
		require.Len(t, w.Accounts, 1)
		require.Equal(t, w.Accounts[0].Label, "testname")
		require.NoError(t, w.Accounts[0].Decrypt("testpass", w.Scrypt))

		t.Run("RemoveAccount", func(t *testing.T) {
			sh := w.Accounts[0].Contract.ScriptHash()
			addr := w.Accounts[0].Address
			e.In.WriteString("y\r")
			e.Run(t, "neo-go", "wallet", "remove",
				"--wallet", walletPath, "--address", addr)
			w, err := wallet.NewWalletFromFile(walletPath)
			require.NoError(t, err)
			require.Nil(t, w.GetAccount(sh))
		})
	})

	t.Run("Import", func(t *testing.T) {
		t.Run("WIF", func(t *testing.T) {
			t.Run("missing wallet", func(t *testing.T) {
				e.RunWithError(t, "neo-go", "wallet", "import")
			})
			priv, err := keys.NewPrivateKey()
			require.NoError(t, err)
			e.In.WriteString("test_account\r")
			e.In.WriteString("qwerty\r")
			e.In.WriteString("qwerty\r")
			e.Run(t, "neo-go", "wallet", "import", "--wallet", walletPath,
				"--wif", priv.WIF())

			w, err := wallet.NewWalletFromFile(walletPath)
			require.NoError(t, err)
			acc := w.GetAccount(priv.GetScriptHash())
			require.NotNil(t, acc)
			require.Equal(t, "test_account", acc.Label)
			require.NoError(t, acc.Decrypt("qwerty", w.Scrypt))

			t.Run("AlreadyExists", func(t *testing.T) {
				e.In.WriteString("test_account_2\r")
				e.In.WriteString("qwerty2\r")
				e.In.WriteString("qwerty2\r")
				e.RunWithError(t, "neo-go", "wallet", "import",
					"--wallet", walletPath, "--wif", priv.WIF())
			})

			t.Run("contract", func(t *testing.T) {
				priv, err = keys.NewPrivateKey()
				require.NoError(t, err)

				t.Run("invalid script", func(t *testing.T) {
					e.In.WriteString("test_account_3\r")
					e.In.WriteString("qwerty\r")
					e.In.WriteString("qwerty\r")
					e.RunWithError(t, "neo-go", "wallet", "import",
						"--wallet", walletPath, "--wif", priv.WIF(), "--contract", "not-a-hex")
				})
				check := func(t *testing.T, expectedLabel string, pass string) {
					w, err := wallet.NewWalletFromFile(walletPath)
					require.NoError(t, err)
					acc := w.GetAccount(priv.GetScriptHash())
					require.NotNil(t, acc)
					require.Equal(t, expectedLabel, acc.Label)
					require.NoError(t, acc.Decrypt(pass, w.Scrypt))
				}
				t.Run("good", func(t *testing.T) {
					e.In.WriteString("test_account_3\r")
					e.In.WriteString("qwerty\r")
					e.In.WriteString("qwerty\r")
					e.Run(t, "neo-go", "wallet", "import",
						"--wallet", walletPath, "--wif", priv.WIF(), "--contract", "0a0b0c")
					check(t, "test_account_3", "qwerty")
				})

				t.Run("from wallet config", func(t *testing.T) {
					tmp := t.TempDir()
					configPath := filepath.Join(tmp, "config.yaml")
					cfg := config.Wallet{
						Path:     walletPath,
						Password: "qwerty",
					}
					res, err := yaml.Marshal(cfg)
					require.NoError(t, err)
					require.NoError(t, os.WriteFile(configPath, res, 0666))
					priv, err = keys.NewPrivateKey()
					require.NoError(t, err)
					e.In.WriteString("test_account_4\r")
					e.Run(t, "neo-go", "wallet", "import",
						"--wallet-config", configPath, "--wif", priv.WIF(), "--contract", "0a0b0c0d")
					check(t, "test_account_4", "qwerty")
				})
				t.Run("from wallet config with account name argument", func(t *testing.T) {
					tmp := t.TempDir()
					configPath := filepath.Join(tmp, "config.yaml")
					cfg := config.Wallet{
						Path:     walletPath,
						Password: "qwerty",
					}
					res, err := yaml.Marshal(cfg)
					require.NoError(t, err)
					require.NoError(t, os.WriteFile(configPath, res, 0666))
					priv, err = keys.NewPrivateKey()
					require.NoError(t, err)
					e.Run(t, "neo-go", "wallet", "import",
						"--wallet-config", configPath, "--wif", priv.WIF(), "--contract", "0a0b0c0d", "--name", "test_account_5")
					check(t, "test_account_5", "qwerty")
				})
			})
		})
		t.Run("EncryptedWIF", func(t *testing.T) {
			acc, err := wallet.NewAccount()
			require.NoError(t, err)
			require.NoError(t, acc.Encrypt("somepass", keys.NEP2ScryptParams()))

			t.Run("InvalidPassword", func(t *testing.T) {
				e.In.WriteString("password1\r")
				e.RunWithError(t, "neo-go", "wallet", "import", "--wallet", walletPath,
					"--wif", acc.EncryptedWIF, "--name", "acc1")
			})

			e.In.WriteString("somepass\r")
			e.Run(t, "neo-go", "wallet", "import", "--wallet", walletPath,
				"--wif", acc.EncryptedWIF, "--name", "acc1")

			w, err := wallet.NewWalletFromFile(walletPath)
			require.NoError(t, err)
			actual := w.GetAccount(acc.PrivateKey().GetScriptHash())
			require.NotNil(t, actual)
			require.Equal(t, "acc1", actual.Label)
			require.NoError(t, actual.Decrypt("somepass", w.Scrypt))
		})
		t.Run("EncryptedWIF with name specified via input", func(t *testing.T) {
			acc, err := wallet.NewAccount()
			require.NoError(t, err)
			require.NoError(t, acc.Encrypt("somepass", keys.NEP2ScryptParams()))

			e.In.WriteString("acc2\r")
			e.In.WriteString("somepass\r")
			e.Run(t, "neo-go", "wallet", "import", "--wallet", walletPath,
				"--wif", acc.EncryptedWIF)

			w, err := wallet.NewWalletFromFile(walletPath)
			require.NoError(t, err)
			actual := w.GetAccount(acc.PrivateKey().GetScriptHash())
			require.NotNil(t, actual)
			require.Equal(t, "acc2", actual.Label)
			require.NoError(t, actual.Decrypt("somepass", w.Scrypt))
		})
		t.Run("EncryptedWIF with wallet config", func(t *testing.T) {
			pass := "somepass"
			check := func(t *testing.T, configPass string, needUserPass bool) {
				acc, err := wallet.NewAccount()
				require.NoError(t, acc.Encrypt(pass, keys.NEP2ScryptParams()))
				configPath := filepath.Join(t.TempDir(), "wallet-config.yaml")
				require.NoError(t, err)
				cfg := &config.Wallet{
					Path:     walletPath,
					Password: configPass,
				}
				bytes, err := yaml.Marshal(cfg)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(configPath, bytes, os.ModePerm))

				if needUserPass {
					e.In.WriteString(pass + "\r")
				}
				e.Run(t, "neo-go", "wallet", "import", "--wallet-config", configPath,
					"--wif", acc.EncryptedWIF, "--name", "acc3"+configPass)

				w, err := wallet.NewWalletFromFile(walletPath)
				require.NoError(t, err)
				actual := w.GetAccount(acc.PrivateKey().GetScriptHash())
				require.NotNil(t, actual)
				require.Equal(t, "acc3"+configPass, actual.Label)
				require.NoError(t, actual.Decrypt(pass, w.Scrypt))
			}
			t.Run("config password mismatch", func(t *testing.T) {
				check(t, pass+"badpass", true)
			})

			t.Run("good config password", func(t *testing.T) {
				check(t, pass, false)
			})
		})
		t.Run("Multisig", func(t *testing.T) {
			t.Run("missing wallet", func(t *testing.T) {
				e.RunWithError(t, "neo-go", "wallet", "import-multisig")
			})
			t.Run("insufficient pubs", func(t *testing.T) {
				e.RunWithError(t, "neo-go", "wallet", "import-multisig",
					"--wallet", walletPath,
					"--min", "2")
			})
			privs, pubs := testcli.GenerateKeys(t, 4)
			cmd := []string{"neo-go", "wallet", "import-multisig",
				"--wallet", walletPath,
				"--min", "2"}
			t.Run("invalid pub encoding", func(t *testing.T) {
				e.RunWithError(t, append(cmd, pubs[1].StringCompressed(),
					pubs[1].StringCompressed(),
					pubs[2].StringCompressed(),
					"not-a-pub")...)
			})
			t.Run("missing WIF", func(t *testing.T) {
				e.RunWithError(t, append(cmd, pubs[0].StringCompressed(),
					pubs[1].StringCompressed(),
					pubs[2].StringCompressed(),
					pubs[3].StringCompressed())...)
			})
			cmd = append(cmd, "--wif", privs[0].WIF())
			t.Run("InvalidPublicKeys", func(t *testing.T) {
				e.In.WriteString("multiacc\r")
				e.In.WriteString("multipass\r")
				e.In.WriteString("multipass\r")
				defer e.In.Reset()

				e.RunWithError(t, append(cmd, pubs[1].StringCompressed(),
					pubs[1].StringCompressed(),
					pubs[2].StringCompressed(),
					pubs[3].StringCompressed())...)
			})
			e.In.WriteString("multiacc\r")
			e.In.WriteString("multipass\r")
			e.In.WriteString("multipass\r")
			e.Run(t, append(cmd, pubs[0].StringCompressed(),
				pubs[1].StringCompressed(),
				pubs[2].StringCompressed(),
				pubs[3].StringCompressed())...)

			script, err := smartcontract.CreateMultiSigRedeemScript(2, pubs)
			require.NoError(t, err)

			w, err := wallet.NewWalletFromFile(walletPath)
			require.NoError(t, err)
			actual := w.GetAccount(hash.Hash160(script))
			require.NotNil(t, actual)
			require.NoError(t, actual.Decrypt("multipass", w.Scrypt))
			require.Equal(t, script, actual.Contract.Script)

			t.Run("double-import", func(t *testing.T) {
				e.In.WriteString("multiacc\r")
				e.In.WriteString("multipass\r")
				e.In.WriteString("multipass\r")
				e.RunWithError(t, append(cmd, pubs[0].StringCompressed(),
					pubs[1].StringCompressed(),
					pubs[2].StringCompressed(),
					pubs[3].StringCompressed())...)
			})

			privs, pubs = testcli.GenerateKeys(t, 3)
			script, err = smartcontract.CreateMultiSigRedeemScript(2, pubs)
			require.NoError(t, err)
			// Create a wallet and import a standard account
			e.Run(t, "neo-go", "wallet", "init", "--wallet", walletPath)
			e.In.WriteString("standardacc\rstdpass\rstdpass\r")
			e.Run(t, "neo-go", "wallet", "import",
				"--wallet", walletPath,
				"--wif", privs[0].WIF())
			w, err = wallet.NewWalletFromFile(walletPath)
			require.NoError(t, err)
			actual = w.GetAccount(privs[0].GetScriptHash())
			require.NotNil(t, actual)
			require.NotEqual(t, actual.Contract.Script, script)

			// Test when a public key of an already imported account is present
			t.Run("existing account public key, no WIF", func(t *testing.T) {
				e.Run(t, "neo-go", "wallet", "import-multisig",
					"--wallet", walletPath,
					"--min", "2",
					pubs[0].StringCompressed(), // Public key of the already imported account
					pubs[1].StringCompressed(),
					pubs[2].StringCompressed())

				w, err := wallet.NewWalletFromFile(walletPath)
				require.NoError(t, err)
				actual := w.GetAccount(hash.Hash160(script))
				require.NotNil(t, actual)
				require.Equal(t, actual.Contract.Script, script)
				require.NoError(t, actual.Decrypt("stdpass", w.Scrypt))
				require.NotEqual(t, actual.Address, w.GetAccount(privs[0].GetScriptHash()).Address)
			})

			// Test when no public key of an already imported account is present, and no WIF is provided
			t.Run("no existing account public key, no WIF", func(t *testing.T) {
				_, pubsNew := testcli.GenerateKeys(t, 3)
				scriptNew, err := smartcontract.CreateMultiSigRedeemScript(2, pubsNew)
				require.NoError(t, err)

				e.RunWithError(t, "neo-go", "wallet", "import-multisig",
					"--wallet", walletPath,
					"--min", "2",
					pubsNew[0].StringCompressed(),
					pubsNew[1].StringCompressed(),
					pubsNew[2].StringCompressed())

				w, err := wallet.NewWalletFromFile(walletPath)
				require.NoError(t, err)
				actual := w.GetAccount(hash.Hash160(scriptNew))
				require.Nil(t, actual)
			})
		})
	})
}

func TestWalletExport(t *testing.T) {
	e := testcli.NewExecutor(t, false)

	t.Run("missing wallet", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "export")
	})
	t.Run("invalid address", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "export",
			"--wallet", testcli.ValidatorWallet, "not-an-address")
	})
	t.Run("Encrypted", func(t *testing.T) {
		e.Run(t, "neo-go", "wallet", "export",
			"--wallet", testcli.ValidatorWallet, testcli.ValidatorAddr)
		line, err := e.Out.ReadString('\n')
		require.NoError(t, err)
		enc, err := keys.NEP2Encrypt(testcli.ValidatorPriv, "one", keys.ScryptParams{N: 2, R: 1, P: 1}) // these params used in validator wallet for better resources consumption
		require.NoError(t, err)
		require.Equal(t, enc, strings.TrimSpace(line))
	})
	t.Run("Decrypted", func(t *testing.T) {
		t.Run("NoAddress", func(t *testing.T) {
			e.RunWithError(t, "neo-go", "wallet", "export",
				"--wallet", testcli.ValidatorWallet, "--decrypt")
		})
		t.Run("EOF reading password", func(t *testing.T) {
			e.RunWithError(t, "neo-go", "wallet", "export",
				"--wallet", testcli.ValidatorWallet, "--decrypt", testcli.ValidatorAddr)
		})
		t.Run("invalid password", func(t *testing.T) {
			e.In.WriteString("invalid_pass\r")
			e.RunWithError(t, "neo-go", "wallet", "export",
				"--wallet", testcli.ValidatorWallet, "--decrypt", testcli.ValidatorAddr)
		})
		e.In.WriteString("one\r")
		e.Run(t, "neo-go", "wallet", "export",
			"--wallet", testcli.ValidatorWallet, "--decrypt", testcli.ValidatorAddr)
		line, err := e.Out.ReadString('\n')
		require.NoError(t, err)
		require.Equal(t, testcli.ValidatorWIF, strings.TrimSpace(line))
	})
}

func TestWalletClaimGas(t *testing.T) {
	e := testcli.NewExecutor(t, true)

	t.Run("missing wallet path", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "claim",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--address", testcli.TestWalletAccount)
	})
	t.Run("missing address", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "claim",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", testcli.TestWalletPath)
	})
	t.Run("invalid address", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "claim",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", testcli.TestWalletPath,
			"--address", util.Uint160{}.StringLE())
	})
	t.Run("missing endpoint", func(t *testing.T) {
		e.In.WriteString("testpass\r")
		e.RunWithError(t, "neo-go", "wallet", "claim",
			"--wallet", testcli.TestWalletPath,
			"--address", testcli.TestWalletAccount)
	})
	t.Run("insufficient funds", func(t *testing.T) {
		e.In.WriteString("testpass\r")
		e.RunWithError(t, "neo-go", "wallet", "claim",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", testcli.TestWalletPath,
			"--address", testcli.TestWalletAccount)
	})

	args := []string{
		"neo-go", "wallet", "nep17", "multitransfer",
		"--rpc-endpoint", "http://" + e.RPC.Addresses()[0],
		"--wallet", testcli.ValidatorWallet,
		"--from", testcli.ValidatorAddr,
		"--force",
		"NEO:" + testcli.TestWalletAccount + ":1000",
		"GAS:" + testcli.TestWalletAccount + ":1000", // for tx send
	}
	e.In.WriteString("one\r")
	e.Run(t, args...)
	e.CheckTxPersisted(t)

	h, err := address.StringToUint160(testcli.TestWalletAccount)
	require.NoError(t, err)

	balanceBefore := e.Chain.GetUtilityTokenBalance(h)
	claimHeight := e.Chain.BlockHeight() + 1
	cl, err := e.Chain.CalculateClaimable(h, claimHeight)
	require.NoError(t, err)
	require.True(t, cl.Sign() > 0)

	e.In.WriteString("testpass\r")
	e.Run(t, "neo-go", "wallet", "claim",
		"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
		"--wallet", testcli.TestWalletPath,
		"--address", testcli.TestWalletAccount,
		"--force")
	tx, height := e.CheckTxPersisted(t)
	balanceBefore.Sub(balanceBefore, big.NewInt(tx.NetworkFee+tx.SystemFee))
	balanceBefore.Add(balanceBefore, cl)

	balanceAfter := e.Chain.GetUtilityTokenBalance(h)
	// height can be bigger than claimHeight especially when tests are executed with -race.
	if height == claimHeight {
		require.Equal(t, 0, balanceAfter.Cmp(balanceBefore))
	} else {
		require.Equal(t, 1, balanceAfter.Cmp(balanceBefore))
	}
	t.Run("await", func(t *testing.T) {
		args := []string{
			"neo-go", "wallet", "nep17", "multitransfer",
			"--rpc-endpoint", "http://" + e.RPC.Addresses()[0],
			"--wallet", testcli.ValidatorWallet,
			"--from", testcli.ValidatorAddr, "--await",
			"--force",
			"NEO:" + testcli.TestWalletAccount + ":1000",
			"GAS:" + testcli.TestWalletAccount + ":1000", // for tx send
		}
		e.In.WriteString("one\r")
		e.Run(t, args...)
		e.CheckAwaitableTxPersisted(t)

		h, err := address.StringToUint160(testcli.TestWalletAccount)
		require.NoError(t, err)

		balanceBefore := e.Chain.GetUtilityTokenBalance(h)
		claimHeight := e.Chain.BlockHeight() + 1
		cl, err := e.Chain.CalculateClaimable(h, claimHeight)
		require.NoError(t, err)
		require.True(t, cl.Sign() > 0)

		e.In.WriteString("testpass\r")
		e.Run(t, "neo-go", "wallet", "claim",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", testcli.TestWalletPath,
			"--address", testcli.TestWalletAccount,
			"--force", "--await")
		tx, height = e.CheckAwaitableTxPersisted(t)
		balanceBefore.Sub(balanceBefore, big.NewInt(tx.NetworkFee+tx.SystemFee))
		balanceBefore.Add(balanceBefore, cl)

		balanceAfter = e.Chain.GetUtilityTokenBalance(h)
		// height can be bigger than claimHeight especially when tests are executed with -race.
		if height == claimHeight {
			require.Equal(t, 0, balanceAfter.Cmp(balanceBefore))
		} else {
			require.Equal(t, 1, balanceAfter.Cmp(balanceBefore))
		}
	})
}

func TestWalletImportDeployed(t *testing.T) {
	tmpDir := t.TempDir()
	e := testcli.NewExecutor(t, true)
	h := deployVerifyContract(t, e)
	walletPath := filepath.Join(tmpDir, "wallet.json")

	e.Run(t, "neo-go", "wallet", "init", "--wallet", walletPath)

	priv, err := keys.NewPrivateKey()
	require.NoError(t, err)

	t.Run("missing wallet", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "import-deployed")
	})
	t.Run("missing contract sh", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "import-deployed",
			"--wallet", walletPath)
	})
	t.Run("missing WIF", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "import-deployed",
			"--wallet", walletPath, "--contract", h.StringLE())
	})
	t.Run("missing endpoint", func(t *testing.T) {
		e.In.WriteString("acc\rpass\rpass\r")
		e.RunWithError(t, "neo-go", "wallet", "import-deployed",
			"--wallet", walletPath, "--contract", h.StringLE(),
			"--wif", priv.WIF())
	})
	t.Run("unknown contract", func(t *testing.T) {
		e.In.WriteString("acc\rpass\rpass\r")
		e.RunWithError(t, "neo-go", "wallet", "import-deployed",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", walletPath, "--contract", util.Uint160{}.StringLE(),
			"--wif", priv.WIF())
	})
	t.Run("no `verify` method", func(t *testing.T) {
		badH := deployNNSContract(t, e) // wrong contract with no `verify` method
		e.In.WriteString("acc\rpass\rpass\r")
		e.RunWithError(t, "neo-go", "wallet", "import-deployed",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", walletPath, "--contract", badH.StringLE(),
			"--wif", priv.WIF())
	})

	e.In.WriteString("acc\rpass\rpass\r")
	e.Run(t, "neo-go", "wallet", "import-deployed",
		"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
		"--wallet", walletPath, "--wif", priv.WIF(),
		"--contract", h.StringLE())

	contractAddr := address.Uint160ToString(h)
	checkDeployed := func(t *testing.T) {
		w, err := wallet.NewWalletFromFile(walletPath)
		require.NoError(t, err)
		require.Equal(t, 1, len(w.Accounts))
		actualAddr := w.Accounts[0].Address
		require.Equal(t, contractAddr, actualAddr)
		require.True(t, w.Accounts[0].Contract.Deployed)
	}
	checkDeployed(t)

	t.Run("re-importing", func(t *testing.T) {
		e.In.WriteString("acc\rpass\rpass\r")
		e.RunWithError(t, "neo-go", "wallet", "import-deployed",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", walletPath, "--wif", priv.WIF(),
			"--contract", h.StringLE())
	})

	t.Run("Sign", func(t *testing.T) {
		e.In.WriteString("one\r")
		e.Run(t, "neo-go", "wallet", "nep17", "multitransfer",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", testcli.ValidatorWallet, "--from", testcli.ValidatorAddr,
			"--force",
			"NEO:"+contractAddr+":10",
			"GAS:"+contractAddr+":10")
		e.CheckTxPersisted(t)

		privTo, err := keys.NewPrivateKey()
		require.NoError(t, err)

		e.In.WriteString("pass\r")
		e.Run(t, "neo-go", "wallet", "nep17", "transfer",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", walletPath, "--from", contractAddr,
			"--force",
			"--to", privTo.Address(), "--token", "NEO", "--amount", "1")
		e.CheckTxPersisted(t)

		b, _ := e.Chain.GetGoverningTokenBalance(h)
		require.Equal(t, big.NewInt(9), b)
		b, _ = e.Chain.GetGoverningTokenBalance(privTo.GetScriptHash())
		require.Equal(t, big.NewInt(1), b)
	})

	t.Run("import with name argument", func(t *testing.T) {
		e.Run(t, "neo-go", "wallet", "remove",
			"--wallet", walletPath, "--address", address.Uint160ToString(h), "--force")
		e.In.WriteString("pass\rpass\r")
		e.Run(t, "neo-go", "wallet", "import-deployed",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", walletPath, "--wif", priv.WIF(),
			"--contract", h.StringLE(), "--name", "acc")
		checkDeployed(t)
	})

	t.Run("import with name argument and wallet config", func(t *testing.T) {
		e.Run(t, "neo-go", "wallet", "remove",
			"--wallet", walletPath, "--address", address.Uint160ToString(h), "--force")
		configPath := filepath.Join(t.TempDir(), "wallet-config.yaml")
		cfg := &config.Wallet{
			Path:     walletPath,
			Password: "pass",
		}
		bytes, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, bytes, os.ModePerm))
		e.Run(t, "neo-go", "wallet", "import-deployed",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet-config", configPath, "--wif", priv.WIF(),
			"--contract", h.StringLE(), "--name", "acc")
		checkDeployed(t)
	})
}

func TestStripKeys(t *testing.T) {
	e := testcli.NewExecutor(t, true)
	tmpDir := t.TempDir()
	walletPath := filepath.Join(tmpDir, "wallet.json")
	e.In.WriteString("acc1\r")
	e.In.WriteString("pass\r")
	e.In.WriteString("pass\r")
	e.Run(t, "neo-go", "wallet", "init", "--wallet", walletPath, "--account")
	w1, err := wallet.NewWalletFromFile(walletPath)
	require.NoError(t, err)

	e.RunWithError(t, "neo-go", "wallet", "strip-keys", "--wallet", walletPath, "something")
	e.RunWithError(t, "neo-go", "wallet", "strip-keys", "--wallet", walletPath+".bad")

	e.In.WriteString("no")
	e.Run(t, "neo-go", "wallet", "strip-keys", "--wallet", walletPath)
	w2, err := wallet.NewWalletFromFile(walletPath)
	require.NoError(t, err)
	require.Equal(t, w1, w2)

	e.In.WriteString("y\r")
	e.Run(t, "neo-go", "wallet", "strip-keys", "--wallet", walletPath)
	e.Run(t, "neo-go", "wallet", "strip-keys", "--wallet", walletPath, "--force") // Does nothing effectively, but tests the force flag.
	w3, err := wallet.NewWalletFromFile(walletPath)
	require.NoError(t, err)
	for _, a := range w3.Accounts {
		require.Equal(t, "", a.EncryptedWIF)
	}
}

func TestOfflineSigning(t *testing.T) {
	e := testcli.NewExecutor(t, true)
	tmpDir := t.TempDir()
	walletPath := filepath.Join(tmpDir, "wallet.json")
	txPath := filepath.Join(tmpDir, "tx.json")

	// Copy wallet.
	w, err := wallet.NewWalletFromFile(testcli.ValidatorWallet)
	require.NoError(t, err)
	jOut, err := w.JSON()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(walletPath, jOut, 0644))

	// And remove keys from it.
	e.Run(t, "neo-go", "wallet", "strip-keys", "--wallet", walletPath, "--force")

	t.Run("1/1 multisig", func(t *testing.T) {
		args := []string{"neo-go", "wallet", "nep17", "transfer",
			"--rpc-endpoint", "http://" + e.RPC.Addresses()[0],
			"--wallet", walletPath,
			"--from", testcli.ValidatorAddr,
			"--to", w.Accounts[0].Address,
			"--token", "NEO",
			"--amount", "1",
			"--force",
		}
		// walletPath has no keys, so this can't be sent.
		e.RunWithError(t, args...)
		// But can be saved.
		e.Run(t, append(args, "--out", txPath)...)
		// It can't be signed with the original wallet.
		e.RunWithError(t, "neo-go", "wallet", "sign",
			"--wallet", walletPath, "--address", testcli.ValidatorAddr,
			"--in", txPath, "--out", txPath)
		t.Run("sendtx", func(t *testing.T) {
			// And it can't be sent.
			e.RunWithError(t, "neo-go", "util", "sendtx",
				"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
				txPath)
			// Even with too many arguments.
			e.RunWithError(t, "neo-go", "util", "sendtx",
				"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
				txPath, txPath)
			// Or no arguments at all.
			e.RunWithError(t, "neo-go", "util", "sendtx",
				"--rpc-endpoint", "http://"+e.RPC.Addresses()[0])
		})
		// But it can be signed with a proper wallet.
		e.In.WriteString("one\r")
		e.Run(t, "neo-go", "wallet", "sign",
			"--wallet", testcli.ValidatorWallet, "--address", testcli.ValidatorAddr,
			"--in", txPath, "--out", txPath)
		// And then anyone can send (even via wallet sign).
		e.Run(t, "neo-go", "wallet", "sign",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", walletPath, "--address", testcli.ValidatorAddr,
			"--in", txPath)
	})
	e.CheckTxPersisted(t)

	t.Run("await 1/1 multisig", func(t *testing.T) {
		args := []string{"neo-go", "wallet", "nep17", "transfer",
			"--rpc-endpoint", "http://" + e.RPC.Addresses()[0],
			"--wallet", walletPath,
			"--from", testcli.ValidatorAddr,
			"--to", w.Accounts[0].Address,
			"--token", "NEO",
			"--amount", "1",
			"--force",
		}
		e.Run(t, append(args, "--out", txPath)...)

		e.In.WriteString("one\r")
		e.Run(t, "neo-go", "wallet", "sign",
			"--wallet", testcli.ValidatorWallet, "--address", testcli.ValidatorAddr,
			"--in", txPath, "--out", txPath)

		e.Run(t, "neo-go", "wallet", "sign",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			"--wallet", walletPath, "--address", testcli.ValidatorAddr,
			"--in", txPath, "--await")
		e.CheckAwaitableTxPersisted(t)
	})

	t.Run("simple signature", func(t *testing.T) {
		simpleAddr := w.Accounts[0].Address
		args := []string{"neo-go", "wallet", "nep17", "transfer",
			"--rpc-endpoint", "http://" + e.RPC.Addresses()[0],
			"--wallet", walletPath,
			"--from", simpleAddr,
			"--to", testcli.ValidatorAddr,
			"--token", "NEO",
			"--amount", "1",
			"--force",
		}
		// walletPath has no keys, so this can't be sent.
		e.RunWithError(t, args...)
		// But can be saved.
		e.Run(t, append(args, "--out", txPath)...)
		// It can't be signed with the original wallet.
		e.RunWithError(t, "neo-go", "wallet", "sign",
			"--wallet", walletPath, "--address", simpleAddr,
			"--in", txPath, "--out", txPath)
		// But can be with a proper one.
		e.In.WriteString("one\r")
		e.Run(t, "neo-go", "wallet", "sign",
			"--wallet", testcli.ValidatorWallet, "--address", simpleAddr,
			"--in", txPath, "--out", txPath)
		// Sending without an RPC node is not likely to succeed.
		e.RunWithError(t, "neo-go", "util", "sendtx", txPath)
		// But it requires no wallet at all.
		e.Run(t, "neo-go", "util", "sendtx",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			txPath)
	})

	t.Run("await simple signature", func(t *testing.T) {
		simpleAddr := w.Accounts[0].Address
		args := []string{"neo-go", "wallet", "nep17", "transfer",
			"--rpc-endpoint", "http://" + e.RPC.Addresses()[0],
			"--wallet", walletPath,
			"--from", simpleAddr,
			"--to", testcli.ValidatorAddr,
			"--token", "NEO",
			"--amount", "1",
			"--force",
		}

		e.Run(t, append(args, "--out", txPath)...)

		e.In.WriteString("one\r")
		e.Run(t, "neo-go", "wallet", "sign",
			"--wallet", testcli.ValidatorWallet, "--address", simpleAddr,
			"--in", txPath, "--out", txPath)

		e.Run(t, "neo-go", "util", "sendtx",
			"--rpc-endpoint", "http://"+e.RPC.Addresses()[0],
			txPath, "--await")
		e.CheckAwaitableTxPersisted(t)
	})
}

func TestWalletDump(t *testing.T) {
	e := testcli.NewExecutor(t, false)

	t.Run("missing wallet", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "dump")
	})
	cmd := []string{"neo-go", "wallet", "dump", "--wallet", testcli.TestWalletPath}
	e.Run(t, cmd...)
	rawStr := strings.TrimSpace(e.Out.String())
	w := new(wallet.Wallet)
	require.NoError(t, json.Unmarshal([]byte(rawStr), w))
	require.Equal(t, 1, len(w.Accounts))
	require.Equal(t, testcli.TestWalletAccount, w.Accounts[0].Address)

	t.Run("with decrypt", func(t *testing.T) {
		cmd = append(cmd, "--decrypt")
		t.Run("EOF reading password", func(t *testing.T) {
			e.RunWithError(t, cmd...)
		})
		t.Run("invalid password", func(t *testing.T) {
			e.In.WriteString("invalidpass\r")
			e.RunWithError(t, cmd...)
		})
		t.Run("good", func(t *testing.T) {
			e.In.WriteString("testpass\r")
			e.Run(t, cmd...)
			rawStr := strings.TrimSpace(e.Out.String())
			w := new(wallet.Wallet)
			require.NoError(t, json.Unmarshal([]byte(rawStr), w))
			require.Equal(t, 1, len(w.Accounts))
			require.Equal(t, testcli.TestWalletAccount, w.Accounts[0].Address)
		})
		t.Run("good, from wallet config", func(t *testing.T) {
			tmp := t.TempDir()
			configPath := filepath.Join(tmp, "config.yaml")
			cfg := config.Wallet{
				Path:     testcli.TestWalletPath,
				Password: "testpass",
			}
			res, err := yaml.Marshal(cfg)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(configPath, res, 0666))
			e.Run(t, "neo-go", "wallet", "dump", "--wallet-config", configPath)
			rawStr := strings.TrimSpace(e.Out.String())
			w := new(wallet.Wallet)
			require.NoError(t, json.Unmarshal([]byte(rawStr), w))
			require.Equal(t, 1, len(w.Accounts))
			require.Equal(t, testcli.TestWalletAccount, w.Accounts[0].Address)
		})
	})
}

func TestWalletDumpKeys(t *testing.T) {
	e := testcli.NewExecutor(t, false)
	t.Run("missing wallet", func(t *testing.T) {
		e.RunWithError(t, "neo-go", "wallet", "dump-keys")
	})
	cmd := []string{"neo-go", "wallet", "dump-keys", "--wallet", testcli.ValidatorWallet}
	pubRegex := "^0[23][a-hA-H0-9]{64}$"
	t.Run("all", func(t *testing.T) {
		e.Run(t, cmd...)
		e.CheckNextLine(t, "Nhfg3TbpwogLvDGVvAvqyThbsHgoSUKwtn")
		e.CheckNextLine(t, pubRegex)
		e.CheckNextLine(t, "^\\s*$")
		e.CheckNextLine(t, "NVTiAjNgagDkTr5HTzDmQP9kPwPHN5BgVq")
		for i := 0; i < 4; i++ {
			e.CheckNextLine(t, pubRegex)
		}
		e.CheckNextLine(t, "^\\s*$")
		e.CheckNextLine(t, "NfgHwwTi3wHAS8aFAN243C5vGbkYDpqLHP")
		e.CheckNextLine(t, pubRegex)
		e.CheckEOF(t)
	})
	t.Run("unknown address", func(t *testing.T) {
		cmd := append(cmd, "--address", util.Uint160{}.StringLE())
		e.RunWithError(t, cmd...)
	})
	t.Run("simple signature", func(t *testing.T) {
		cmd := append(cmd, "--address", "Nhfg3TbpwogLvDGVvAvqyThbsHgoSUKwtn")
		e.Run(t, cmd...)
		e.CheckNextLine(t, "simple signature contract")
		e.CheckNextLine(t, pubRegex)
		e.CheckEOF(t)
	})
	t.Run("3/4 multisig", func(t *testing.T) {
		cmd := append(cmd, "-a", "NVTiAjNgagDkTr5HTzDmQP9kPwPHN5BgVq")
		e.Run(t, cmd...)
		e.CheckNextLine(t, "3 out of 4 multisig contract")
		for i := 0; i < 4; i++ {
			e.CheckNextLine(t, pubRegex)
		}
		e.CheckEOF(t)
	})
	t.Run("1/1 multisig", func(t *testing.T) {
		cmd := append(cmd, "--address", "NfgHwwTi3wHAS8aFAN243C5vGbkYDpqLHP")
		e.Run(t, cmd...)
		e.CheckNextLine(t, "1 out of 1 multisig contract")
		e.CheckNextLine(t, pubRegex)
		e.CheckEOF(t)
	})
}

// Testcase is the wallet of privnet validator.
func TestWalletConvert(t *testing.T) {
	tmpDir := t.TempDir()
	e := testcli.NewExecutor(t, false)

	outPath := filepath.Join(tmpDir, "wallet.json")
	cmd := []string{"neo-go", "wallet", "convert"}
	t.Run("missing wallet", func(t *testing.T) {
		e.RunWithError(t, cmd...)
	})
	cmd = append(cmd, "--wallet", "testdata/testwallet_NEO2.json")
	t.Run("missing out path", func(t *testing.T) {
		e.RunWithError(t, cmd...)
	})
	t.Run("invalid out path", func(t *testing.T) {
		dir := t.TempDir()
		e.RunWithError(t, append(cmd, "--out", dir)...)
	})

	cmd = append(cmd, "--out", outPath)
	t.Run("invalid password", func(t *testing.T) {
		// missing password
		e.RunWithError(t, cmd...)
		// invalid password
		e.In.WriteString("two\r")
		e.RunWithError(t, cmd...)
	})

	// 2 accounts.
	e.In.WriteString("one\r")
	e.In.WriteString("one\r")
	e.Run(t, "neo-go", "wallet", "convert",
		"--wallet", "testdata/testwallet_NEO2.json",
		"--out", outPath)

	actual, err := wallet.NewWalletFromFile(outPath)
	require.NoError(t, err)
	expected, err := wallet.NewWalletFromFile("testdata/testwallet_NEO3.json")
	require.NoError(t, err)
	require.Equal(t, len(actual.Accounts), len(expected.Accounts))
	for _, exp := range expected.Accounts {
		addr, err := address.StringToUint160(exp.Address)
		require.NoError(t, err)

		act := actual.GetAccount(addr)
		require.NotNil(t, act)
		require.Equal(t, exp, act)
	}
}

func deployNNSContract(t *testing.T, e *testcli.Executor) util.Uint160 {
	return testcli.DeployContract(t, e, "../../examples/nft-nd-nns/", "../../examples/nft-nd-nns/nns.yml", testcli.ValidatorWallet, testcli.ValidatorAddr, testcli.ValidatorPass)
}
