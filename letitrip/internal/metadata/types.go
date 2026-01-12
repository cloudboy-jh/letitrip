package metadata

type Track struct {
	Number int
	Title  string
}

type Release struct {
	Artist      string
	Album       string
	Year        string
	Genre       string
	AlbumArtist string
	Tracks      []Track
	CoverArt    []byte
	ReleaseID   string
}
