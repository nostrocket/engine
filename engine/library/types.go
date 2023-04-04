package library

type Wallet struct {
	PrivateKey string
	SeedWords  string
	Account    Account
}

type Account = string

type RocketID = string

type Sha256 = string
