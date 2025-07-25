package sv1

// PETTI - Go Sally Protocol for Exchanging Technical Tasks and Information

type PettiRequest struct {
	PettiVer    string `json:"PettiVer"`
	PackageType struct {
		Request string `json:"Request"`
	} `json:"PackageType"`
	Payload map[string]any `json:"Payload"`
}

type PettiResponse struct {
	PettiVer    string `json:"PettiVer"`
	PackageType struct {
		AnswerOf string `json:"AnswerOf"`
	} `json:"PackageType"`
	ResponsibleAgentUUID string         `json:"ResponsibleAgentUUID"`
	Payload              map[string]any `json:"Payload"`
}
