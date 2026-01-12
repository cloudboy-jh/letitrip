package disc

type TrackTOC struct {
	Number    int
	StartLBA  int
	LengthLBA int
}

type TOC struct {
	Tracks     []TrackTOC
	LeadoutLBA int
}
