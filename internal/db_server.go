package audiostrike

import (
	"database/sql"
	art "github.com/audiostrike/music/pkg/art"
	"log"
	"path/filepath"
)

// NewServer creates a new DbServer to save and serve art through an AustkDb database.
func NewDbServer(db *AustkDb, artRootPath string) *DbServer {
	log.Printf("artDb: %v", db)
	return &DbServer{db: db, artRootPath: artRootPath}
}

type DbServer struct {
	db *AustkDb
	artRootPath string
}

// Store and get artist info.
func (dbServer *DbServer) StoreArtist(artist *art.Artist, publisher Publisher) (*art.ArtResources, error) {
	err := publisher.VerifyArtist(artist)
	if err != nil {
		return nil, err
	}
	
	err = dbServer.db.PutArtist(artist)
	if err != nil {
		return nil, err
	}
	
	return &art.ArtResources{
	}, nil
}

func (dbServer *DbServer) Artists() (map[string]*art.Artist, error) {
	return dbServer.db.SelectAllArtists()
}

func (dbServer *DbServer) Artist(artistID string) (*art.Artist, error) {
	return dbServer.db.SelectArtist(artistID)
}

// Store and get optional album info.
func (dbServer *DbServer) StoreAlbum(album *art.Album) error {
	return dbServer.db.PutAlbum(album)
}

func (dbServer *DbServer) Albums(artistID string) (map[string]*art.Album, error) {
	return dbServer.db.SelectArtistAlbums(artistID)
}

// Store and get track info.
func (dbServer *DbServer) StoreTrack(track *art.Track, publisher Publisher) error {
	// legacy store doesn't verify track with publisher
	return dbServer.db.PutTrack(track)
}

func (dbServer *DbServer) StoreTrackPayload(track *art.Track, bytes []byte) error {
	// Temporary hack to decouple from DB
	fileServer, err := NewFileServer(dbServer.artRootPath)
	if err != nil {
		log.Printf("StoreTrackPayload to %s error: %v", dbServer.artRootPath, err)
	}
	return fileServer.StoreTrackPayload(track, bytes)
}

func (dbServer *DbServer) TrackFilePath(track *art.Track) string {
	return filepath.Join(dbServer.artRootPath, track.ArtistId, track.ArtistTrackId+".mp3")
}

func (dbServer *DbServer) Tracks(artistID string) (map[string]*art.Track, error) {
	return dbServer.db.SelectArtistTracks(artistID)
}

func (dbServer *DbServer) Track(artistId string, trackId string) (*art.Track, error) {
	return dbServer.db.SelectTrack(artistId, trackId)
}

func (dbServer *DbServer) StorePeer(peer *art.Peer, publisher Publisher) error {
	// this legacy repository doesn't validate the peer with publisher
	return dbServer.db.PutPeer(peer)
}

func (dbServer *DbServer) Peer(pubkey string) (*art.Peer, error) {
	return dbServer.db.SelectPeer(pubkey)
}

func (dbServer *DbServer) Peers() (map[string]*art.Peer, error) {
	peers, err := dbServer.db.SelectAllPeers()
	if err == sql.ErrNoRows {
		return nil, ErrPeerNotFound
	}
	return peers, err
}

