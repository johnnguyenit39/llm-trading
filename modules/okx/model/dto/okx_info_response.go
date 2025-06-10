package model

// OkxInfoResponse represents the response structure for OKX account information
type OkxInfoResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		AdjEq                 string          `json:"adjEq"`
		AvailEq               string          `json:"availEq"`
		BorrowFroz            string          `json:"borrowFroz"`
		Details               []AccountDetail `json:"details"`
		Imr                   string          `json:"imr"`
		IsoEq                 string          `json:"isoEq"`
		MgnRatio              string          `json:"mgnRatio"`
		Mmr                   string          `json:"mmr"`
		NotionalUsd           string          `json:"notionalUsd"`
		NotionalUsdForBorrow  string          `json:"notionalUsdForBorrow"`
		NotionalUsdForFutures string          `json:"notionalUsdForFutures"`
		NotionalUsdForOption  string          `json:"notionalUsdForOption"`
		NotionalUsdForSwap    string          `json:"notionalUsdForSwap"`
		OrdFroz               string          `json:"ordFroz"`
		TotalEq               string          `json:"totalEq"`
		UTime                 string          `json:"uTime"`
		Upl                   string          `json:"upl"`
	} `json:"data"`
}

// AccountDetail represents detailed account information for a specific currency
type AccountDetail struct {
	AccAvgPx              string `json:"accAvgPx"`
	AvailBal              string `json:"availBal"`
	AvailEq               string `json:"availEq"`
	BorrowFroz            string `json:"borrowFroz"`
	CashBal               string `json:"cashBal"`
	Ccy                   string `json:"ccy"`
	ClSpotInUseAmt        string `json:"clSpotInUseAmt"`
	ColBorrAutoConversion string `json:"colBorrAutoConversion"`
	CollateralEnabled     bool   `json:"collateralEnabled"`
	CollateralRestrict    bool   `json:"collateralRestrict"`
	CrossLiab             string `json:"crossLiab"`
	DisEq                 string `json:"disEq"`
	Eq                    string `json:"eq"`
	EqUsd                 string `json:"eqUsd"`
	FixedBal              string `json:"fixedBal"`
	FrozenBal             string `json:"frozenBal"`
	Imr                   string `json:"imr"`
	Interest              string `json:"interest"`
	IsoEq                 string `json:"isoEq"`
	IsoLiab               string `json:"isoLiab"`
	IsoUpl                string `json:"isoUpl"`
	Liab                  string `json:"liab"`
	MaxLoan               string `json:"maxLoan"`
	MaxSpotInUse          string `json:"maxSpotInUse"`
	MgnRatio              string `json:"mgnRatio"`
	Mmr                   string `json:"mmr"`
	NotionalLever         string `json:"notionalLever"`
	OpenAvgPx             string `json:"openAvgPx"`
	OrdFrozen             string `json:"ordFrozen"`
	RewardBal             string `json:"rewardBal"`
	SmtSyncEq             string `json:"smtSyncEq"`
	SpotBal               string `json:"spotBal"`
	SpotCopyTradingEq     string `json:"spotCopyTradingEq"`
	SpotInUseAmt          string `json:"spotInUseAmt"`
	SpotIsoBal            string `json:"spotIsoBal"`
	SpotUpl               string `json:"spotUpl"`
	SpotUplRatio          string `json:"spotUplRatio"`
	StgyEq                string `json:"stgyEq"`
	TotalPnl              string `json:"totalPnl"`
	TotalPnlRatio         string `json:"totalPnlRatio"`
	Twap                  string `json:"twap"`
	UTime                 string `json:"uTime"`
	Upl                   string `json:"upl"`
	UplLiab               string `json:"uplLiab"`
}
