package types

type Logs struct {
	Stdout []string `json:"stdout"`
	Stderr []string `json:"stderr"`
}
