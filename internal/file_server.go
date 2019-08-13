package audiostrike

import (
	"fmt"
	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

type FileServer struct {
	rootPath string
	// peers indexed by pubkey
	peers map[string]*art.Peer
	// artists indexed by ArtistId
	artists map[string]*art.Artist
	// albums indexed by ArtistId then by ArtistAlbumId
	albums map[string]map[string]*art.Album
	// tracks indexed by ArtistId then by ArtistTrackId
	tracks map[string]map[string]*art.Track
	// tracks indexed by ArtistId then by ArtistAlbumId then by AlbumTrackNumber
	albumTracks map[string]map[string]map[uint32]*art.Track
}

// NewFileServer creates a new FileServer to save and serve art in sudirectories of artDirPath.
func NewFileServer(artDirPath string) (*FileServer, error) {
	const logPrefix = "NewFileServer "

	fileServer := FileServer{
		rootPath:    artDirPath,
		artists:     make(map[string]*art.Artist),
		tracks:      make(map[string]map[string]*art.Track),
		albums:      make(map[string]map[string]*art.Album),
		albumTracks: make(map[string]map[string]map[uint32]*art.Track),
		peers:       make(map[string]*art.Peer),
	}

	err := filepath.Walk(artDirPath, fileServer.readArtFile)
	if err != nil {
		log.Fatalf(logPrefix+"Failed to read art directory, error: %v", err)
		return nil, err
	}
	return &fileServer, nil
}

func (fileServer *FileServer) readArtFile(path string, fileInfo os.FileInfo, err error) error {
	const logPrefix = "FileServer readArtFile "

	// Read the file into an art.Peer
	// ArtistTrackID may be simple alphanumeric identifier
	// or it may optionally include album or other slash-separated hierarchy.
	// Optionally order tracks and albums with numeric prefixes.
	log.Printf(logPrefix+"path: %s", path)
	artArtistPrefixedRegex := regexp.MustCompile("^(?P<root>.*)/(?P<ArtistID>[0-9a-z]+)")
	matchGroups := artArtistPrefixedRegex.FindStringSubmatch(path)
	if len(matchGroups) < 3 {
		log.Printf(logPrefix+"skip non-artist file %s", path)
		return nil
	}
	artArtistDirRegex := regexp.MustCompile("^(?P<root>.*)/(?P<ArtistID>[0-9a-z]+)$")
	matchGroups = artArtistDirRegex.FindStringSubmatch(path)
	if len(matchGroups) == 3 {
		// artist dir
		log.Printf(logPrefix+"matched {full: %s, root: %s, artistId: %s}",
			matchGroups[0], matchGroups[1], matchGroups[2])
		return nil
	}

	artArtistPubRegex, err := regexp.Compile("^(?P<root>.*)/(?P<ArtistID>[0-9a-z]+)/(?P<Pubkey>[0-9a-f]+)[.]pub$")
	if err != nil {
		log.Printf(logPrefix+"pub error: %v", err)
		return err
	}
	matchGroups = artArtistPubRegex.FindStringSubmatch(path)
	if len(matchGroups) == 4 {
		// pub file
		artistID := matchGroups[2]
		pubkey := matchGroups[3]
		log.Printf(logPrefix+"matched {full: %s, root: %s, artistId: %s, pubkey: %s}",
			matchGroups[0], artistID, matchGroups[2], pubkey)
		fileServer.readArtistPub(artistID, pubkey, path)
	}

	artArtistTrackMp3Regex, err := regexp.Compile("^(?P<root>.*)/(?P<ArtistID>[0-9a-z]+)/(?P<ArtistTrackID>[0-9a-z/]+)[.]mp3$")
	if err != nil {
		log.Printf(logPrefix+"error: %v", err)
		return err
	}
	matchGroups = artArtistTrackMp3Regex.FindStringSubmatch(path)
	if len(matchGroups) == 4 {
		// mp3 file
		log.Printf(logPrefix+"matched {full: %s, root: %s, artistId: %s, artistTrackId: %s}",
			matchGroups[0], matchGroups[1], matchGroups[2], matchGroups[3])
	}
	// TODO: If ArtistTrackId is compound, spit it into ArtistAlbumId/AlbumTrackNumber.SimpleTrackId
	return nil
}

func (fileServer *FileServer) readArtistPub(artistID string, pubkey string, artistPubPath string) (*art.Artist, error) {
	const logPrefix = "FileServer readArtistDir "

	artistData, err := ioutil.ReadFile(artistPubPath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile %s error: %v", artistPubPath, err)
		return nil, err
	}

	artist := art.Artist{}
	err = proto.Unmarshal(artistData, &artist)
	// TODO: validate that the pubkey was used for the signature or reject this artist pub file
	fileServer.artists[artistID] = &artist
	return &artist, nil
}

func (fileServer *FileServer) readPeerFile(artDirPath string, pubkey string) error {
	const logPrefix = "FileServer readPeerFile "

	//pubkeyAtHostColonPortRegex, err := regexp.Compile("^([0-9a-f]{66})@([0-9a-z\\-]+):[0-9]+$")
	peerFilePath := path.Join(artDirPath, pubkey)
	peerData, err := ioutil.ReadFile(peerFilePath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile %s error: %v", peerFilePath, err)
		return err
	}
	peer := art.Peer{}
	fileServer.peers[pubkey] = &peer
	//peer := &fileServer.peers[pubkey]
	err = proto.Unmarshal(peerData, fileServer.peers[pubkey])
	return err
}

func (fileServer *FileServer) StoreAlbum(album *art.Album) error {
	return fmt.Errorf("not implemented")
}

func (fileServer *FileServer) Albums(artistId string) (map[string]*art.Album, error) {
	albums, found := fileServer.albums[artistId]
	if !found {
		// Read album info from file system.
		fmt.Errorf("not implemented")
	}

	return albums, nil
}

func (filesServer *FileServer) StoreArtist(artist *art.Artist) error {
	filesServer.artists[artist.ArtistId] = artist
	// TODO: write record to file system
	return nil
}

func (filesServer *FileServer) Artists() (map[string]*art.Artist, error) {
	return filesServer.artists, nil
}

func (fileServer *FileServer) Artist(artistID string) (*art.Artist, error) {
	artist := fileServer.artists[artistID]
	if artist == nil {
		return nil, ErrArtNotFound
	}
	return artist, nil
}

func (filesServer *FileServer) Tracks(artistID string) (map[string]*art.Track, error) {
	return filesServer.tracks[artistID], nil
}

func (fileServer *FileServer) AlbumTracks(artistID string, albumID string) (map[uint32]*art.Track, error) {
	albumTracksForArtist := fileServer.albumTracks[artistID]
	if albumTracksForArtist == nil {
		return nil, ErrArtNotFound
	}
	tracksForArtistAlbum := albumTracksForArtist[albumID]
	if tracksForArtistAlbum == nil {
		return nil, ErrArtNotFound
	}
	return tracksForArtistAlbum, nil
}

// Store and get network peers.

func (fileServer *FileServer) StorePeer(peer *art.Peer) error {
	fileServer.peers[peer.Pubkey] = peer
	// TODO: write to file system
	return nil
}

func (fileServer *FileServer) Peer(pubkey string) (*art.Peer, error) {
	peer := fileServer.peers[pubkey]
	if peer == nil {
		return nil, ErrPeerNotFound
	}
	return peer, nil
}

func (fileServer *FileServer) Peers() (map[string]*art.Peer, error) {
	return fileServer.peers, nil
}

// Store and get track info.
func (fileServer *FileServer) StoreTrack(track *art.Track) error {
	tracksForArtist := fileServer.tracks[track.ArtistId]
	if tracksForArtist == nil {
		tracksForArtist = make(map[string]*art.Track)
		fileServer.tracks[track.ArtistId] = tracksForArtist
	}
	tracksForArtist[track.ArtistTrackId] = track
	if track.ArtistAlbumId != "" || track.AlbumTrackNumber > 0 {
		albumTracksForArtist := fileServer.albumTracks[track.ArtistId]
		if albumTracksForArtist == nil {
			albumTracksForArtist = make(map[string]map[uint32]*art.Track)
			fileServer.albumTracks[track.ArtistId] = albumTracksForArtist
		}
		tracksInArtistAlbum := albumTracksForArtist[track.ArtistAlbumId]
		if tracksInArtistAlbum == nil {
			tracksInArtistAlbum = make(map[uint32]*art.Track)
			albumTracksForArtist[track.ArtistAlbumId] = tracksInArtistAlbum
		}
		tracksInArtistAlbum[track.AlbumTrackNumber] = track
	}
	return nil
}

func (fileServer *FileServer) StoreTrackPayload(artistID string, artistTrackID string, payload []byte) error {
	mp3FilePath := path.Join(fileServer.rootPath, artistID, artistTrackID+".mp3")
	containerDirectory := filepath.Dir(mp3FilePath)

	err := os.MkdirAll(containerDirectory, 0755)
	if err != nil {
		log.Printf("Failed to make directory %s, error: %v", containerDirectory, err)
		return err
	}

	err = ioutil.WriteFile(mp3FilePath, payload, 0644)
	return err
}

func (fileServer *FileServer) Track(artistID string, trackID string) (*art.Track, error) {
	return fileServer.tracks[artistID][trackID], nil
}

func (fileServer *FileServer) TrackFilePath(artistID string, artistTrackID string) string {
	return BuildMp3Filename(fileServer.rootPath, artistID, artistTrackID)
}
