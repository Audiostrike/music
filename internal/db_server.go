package audiostrike

import (
	"database/sql"
	art "github.com/audiostrike/music/pkg/art"
	"log"
)

// NewServer creates a new DbServer to save and serve art through an AustkDb database.
func NewDbServer(db *AustkDb) *DbServer {
	log.Printf("artDb: %v", db)
	return &DbServer{db: db}
}

type DbServer struct {
	db *AustkDb
}

func (dbServer DbServer) Albums(artistId string) (map[string]*art.Album, error) {
	return dbServer.db.SelectArtistAlbums(artistId)
}

func (dbServer DbServer) Artists() (map[string]*art.Artist, error) {
	return dbServer.db.SelectAllArtists()
}

func (dbServer DbServer) Tracks(artistID string) (map[string]*art.Track, error) {
	return dbServer.db.SelectArtistTracks(artistID)
}

func (dbServer DbServer) Peer(pubkey string) (*art.Peer, error) {
	return dbServer.db.SelectPeer(pubkey)
}

func (dbServer DbServer) Peers() (map[string]*art.Peer, error) {
	peers, err := dbServer.db.SelectAllPeers()
	if err == sql.ErrNoRows {
		return nil, ErrPeerNotFound
	}
	return peers, err
}

func (dbServer DbServer) SetArtist(artist *art.Artist) error {
	return dbServer.db.PutArtist(artist)
}

func (dbServer DbServer) SetPeer(peer *art.Peer) error {
	return dbServer.db.PutPeer(peer)
}

func (dbServer DbServer) Track(artistId string, trackId string) (*art.Track, error) {
	return dbServer.db.SelectTrack(artistId, trackId)
}
