package library

type Wallet struct {
	PrivateKey string
	SeedWords  string
	Account    Account
}

type Account = string

type RocketID = string

type Sha256 = string

type Sha1 = string

type Profile struct {
	Name         string `json:"name"`
	Picture      string `json:"picture"`
	About        string `json:"about"`
	Website      string `json:"website"`
	Banner       string `json:"banner"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	DisplayName1 string `json:"displayName"`
	Lud06        string `json:"lud06"`
	Lud16        string `json:"lud16"`
	Nip05        string `json:"nip05"`
	Nip05Valid   bool   `json:"nip05valid"`
}
