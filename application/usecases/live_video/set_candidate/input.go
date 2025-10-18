package set_candidate_usecase

type Message struct {
	Candidate     string
	SDPMid        *string
	SDPMLineIndex *uint16
}
