```
package tokenchain // import "github.com/hectorchu/nano-token-protocol/tokenchain"


TYPES

type Chain struct {
	// Has unexported fields.
}
    Chain represents a token chain.

func LoadChain(address, rpcURL string) (c *Chain, err error)
    LoadChain loads a chain at an address.

func NewChain(rpcURL string) (c *Chain, err error)
    NewChain initializes a new chain.

func NewChainFromSeed(seed []byte, rpcURL string) (c *Chain, err error)
    NewChainFromSeed initializes a new chain from a seed.

func (c *Chain) Address() string
    Address returns the address of the chain.

func (c *Chain) Parse() (err error)
    Parse parses the chain for tokens.

func (c *Chain) SaveState(db *sql.DB) (err error)
    SaveState saves the chain state to the DB.

func (c *Chain) Swap(hash rpc.BlockHash) (s *Swap, err error)
    Swap gets the swap at the specified block hash.

func (c *Chain) Token(hash rpc.BlockHash) (t *Token, err error)
    Token gets the token at the specified block hash.

func (c *Chain) Tokens() (tokens map[string]*Token)
    Tokens gets the chain's tokens.

func (c *Chain) WaitForOpen() (err error)
    WaitForOpen waits for the open block.

type Swap struct {
	// Has unexported fields.
}
    Swap represents a token swap.

func ProposeSwap(c *Chain, a *wallet.Account, counterparty string, t *Token, amount *big.Int) (s *Swap, err error)
    ProposeSwap proposes a swap on-chain.

func (s *Swap) Accept(a *wallet.Account, t *Token, amount *big.Int) (hash rpc.BlockHash, err error)
    Accept accepts a swap proposal.

func (s *Swap) Active() bool
    Active returns whether the swap is active.

func (s *Swap) Cancel(a *wallet.Account) (hash rpc.BlockHash, err error)
    Cancel cancels a swap proposal.

func (s *Swap) Confirm(a *wallet.Account) (hash rpc.BlockHash, err error)
    Confirm confirms a swap proposal.

func (s *Swap) Hash() rpc.BlockHash
    Hash returns the block hash of the swap.

func (s *Swap) Left() (sl SwapLeg)
    Left returns the left leg of the swap.

func (s *Swap) Right() (sl SwapLeg)
    Right returns the right leg of the swap.

type SwapLeg struct {
	Account string
	Token   *Token
	Amount  *big.Int
}
    SwapLeg represents a leg of the swap.

type Token struct {
	// Has unexported fields.
}
    Token represents a token.

func TokenGenesis(c *Chain, a *wallet.Account, name string, supply *big.Int, decimals byte) (t *Token, err error)
    TokenGenesis initializes a new token on a chain.

func (t *Token) Balance(account string) (balance *big.Int)
    Balance gets the balance for account.

func (t *Token) Balances() (balances map[string]*big.Int)
    Balances gets the token balances.

func (t *Token) Decimals() byte
    Decimals returns the token decimals.

func (t *Token) Hash() rpc.BlockHash
    Hash returns the block hash of the token.

func (t *Token) Name() string
    Name returns the token name.

func (t *Token) Supply() *big.Int
    Supply returns the token supply.

func (t *Token) Transfer(a *wallet.Account, account string, amount *big.Int) (hash rpc.BlockHash, err error)
    Transfer transfers an amount of tokens to another account.
```
