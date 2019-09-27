package audiostrike

import (
	"bytes"
	"fmt"
	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

const (
	hexValueRegex = "[0-9a-f]+"
	// simpleIDRegex selects a simple lower-case string for use as an ID (ArtistID, etc.), no spaces or punctuation, safe for URLs, file names, etc.
	simpleIDRegex = "[a-z0-9.-]+"
	// hierarchyRegex selects a series of one or more simple IDs separated by slashes
	hierarchyRegex = simpleIDRegex + "(?:/" + simpleIDRegex + ")*"
)

var (
	artistDirRegexp      *regexp.Regexp = regexp.MustCompile("^/(?P<ArtistID>" + simpleIDRegex + ")$")
	artistFileRegexp     *regexp.Regexp = regexp.MustCompile("^/(?P<ArtistID>" + simpleIDRegex + ")/(?P<file>" + hierarchyRegex + ")$")
	artistArtFileRegexp  *regexp.Regexp = regexp.MustCompile("^/(?P<ArtistID>" + simpleIDRegex + ")/[.]art$")
	artistPubFileRegexp  *regexp.Regexp = regexp.MustCompile("^/(?P<ArtistID>" + simpleIDRegex + ")/(?P<Pubkey>" + hexValueRegex + ")[.]pub$")
	artistTrackMp3Regexp *regexp.Regexp = regexp.MustCompile("^/(?P<ArtistID>" + simpleIDRegex + ")/(?P<ArtistTrackID>" + hierarchyRegex + ")[.]mp3$")
	albumDirRegexp       *regexp.Regexp = regexp.MustCompile("^/(?P<ArtistID>" + simpleIDRegex + ")/(?P<album>" + hierarchyRegex + ")$")
	albumFileRegexp      *regexp.Regexp = regexp.MustCompile("^/(?P<ArtistID>" + simpleIDRegex + ")/(?P<album>" + hierarchyRegex + ")/(?P<file>" + simpleIDRegex + ")$")
)

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

	_ = os.MkdirAll(artDirPath, 0755)

	err := filepath.Walk(artDirPath, fileServer.readFile)
	if err != nil {
		log.Fatalf(logPrefix+"Failed to read art directory, error: %v", err)
		return nil, err
	}
	return &fileServer, nil
}

func (fileServer *FileServer) readFile(prefixedPath string, fileInfo os.FileInfo, err error) error {
	const logPrefix = "FileServer readFile "

	if !strings.HasPrefix(prefixedPath, fileServer.rootPath) {
		log.Printf(logPrefix+"path %s lacks expected prefix %s", prefixedPath, fileServer.rootPath)
		return filepath.SkipDir
	}
	relativePath := prefixedPath[len(fileServer.rootPath):]

	if fileInfo.IsDir() {
		if relativePath == "" {
			log.Printf(logPrefix+"processing root path %s", prefixedPath)
			return nil
		}
		if artistDirRegexp.MatchString(relativePath) {
			artistDirMatchGroups := artistDirRegexp.FindStringSubmatch(relativePath)
			artistID := artistDirMatchGroups[1]
			log.Printf(logPrefix+"processing artist %s dir %s", artistID, prefixedPath)
			return nil
		}

		if albumDirRegexp.MatchString(relativePath) {
			albumDirMatchGroups := albumDirRegexp.FindStringSubmatch(relativePath)
			artistID := albumDirMatchGroups[1]
			artistAlbumID := albumDirMatchGroups[2]
			log.Printf(logPrefix+"processing dir %s for artist %s, album %s", prefixedPath, artistID, artistAlbumID)
			return nil
		}
		log.Printf(logPrefix+"unexpected directory %s does not look like an artist or album directory", prefixedPath)
		return filepath.SkipDir
	}

	// All the files to read are owned by the Artist whose artistID is the file's directory name.
	// Identify that artistID by the name of the directory.
	if !artistFileRegexp.MatchString(relativePath) {
		log.Printf(logPrefix+"skip non-artist file: %s", prefixedPath)
		return nil
	}
	artistFileMatchGroups := artistFileRegexp.FindStringSubmatch(relativePath)
	artistID := artistFileMatchGroups[1]
	log.Printf(logPrefix+"reading file %s for artist %s", relativePath, artistID)

	// if this is the [pubkey].pub file
	if artistPubFileRegexp.MatchString(relativePath) {
		pubFileMatchGroups := artistPubFileRegexp.FindStringSubmatch(relativePath)
		// This is the artist's [pubkey].pub file. Read the publication into art resources.
		pubFileArtistID := pubFileMatchGroups[1]
		if pubFileArtistID != artistID { // sanity check
			return fmt.Errorf("unexpected ArtistID %s in %s for artist %s",
				pubFileArtistID, prefixedPath, artistID)
		}
		pubkey := pubFileMatchGroups[2]
		log.Printf(logPrefix+"pub file %s for artistId: %s, pubkey: %s}", prefixedPath, artistID, pubkey)

		publication, err := fileServer.readPublication(artistID, pubkey, prefixedPath)
		if err != nil {
			log.Fatalf("failed to read artist %s publication %s, error: %v", artistID, prefixedPath, err)
			return err
		}
		resources, err := read(publication)
		if err != nil {
			log.Printf(logPrefix+"failed to read resources from publication, error: %v", err)
			return err
		}
		err = fileServer.indexResources(resources)
		if err != nil {
			log.Printf(logPrefix+"failed to index resources from publication, error: %v", err)
			return err
		}

		log.Printf(logPrefix+"read and indexed resources from publication at %s", relativePath)
		return nil
	}

	// if this is the .art file
	if artistArtFileRegexp.MatchString(relativePath) {
		artFileMatchGroups := artistArtFileRegexp.FindStringSubmatch(relativePath)
		// This is the artist's .art file. Read its art records.
		artFileArtistID := artFileMatchGroups[1]
		if artFileArtistID != artistID { // sanity check
			return fmt.Errorf("unexpected ArtistID %s in %s for artist %s",
				artFileArtistID, prefixedPath, artistID)
		}

		err := fileServer.readArtFile(artistID, prefixedPath)
		if err != nil {
			log.Fatalf("failed to read artist %s resources %s, error: %v", artistID, prefixedPath, err)
			return err
		}

		return nil
	}

	// Read it as art.ArtResources
	// ArtistTrackID may be simple alphanumeric identifier
	// or it may optionally include album or other slash-separated hierarchy.
	// Optionally order tracks and albums with numeric prefixes.

	// Finally, check whether this is an .mp3 file published by the artist.
	if artistTrackMp3Regexp.MatchString(relativePath) {
		artistTrackMp3MatchGroups := artistTrackMp3Regexp.FindStringSubmatch(relativePath)
		// mp3 file
		trackID := artistTrackMp3MatchGroups[2]
		log.Printf(logPrefix+"matched mp3 %s as track ID %s for %s",
			prefixedPath, trackID, artistID)
	} else {
		return fmt.Errorf("Unknown file type: %s", prefixedPath)
	}
	return nil
}

// savePublishedResources saves an .art file with the given resources and a .pub file with the given publication.
func (fileServer *FileServer) savePublishedResources(publication *art.ArtistPublication, resources *art.ArtResources) error {
	const logPrefix = "FileServer savePublishedResources "

	// Ensure that this artist has a directory.
	artistDirectory := filepath.Join(fileServer.rootPath, publication.Artist.ArtistId)
	_ = os.MkdirAll(artistDirectory, 0755)

	// Write the publication's serialized resources to the .art file
	artPath := fileServer.artPath(publication.Artist)
	_, err := os.Stat(artPath)
	if os.IsNotExist(err) {
		log.Printf(logPrefix+"publishing %v to %s", resources, artPath)
	} else {
		log.Printf(logPrefix+"republishing %v to %s", resources, artPath)
	}

	err = ioutil.WriteFile(artPath, publication.SerializedArtResources, 0644)
	if err != nil {
		log.Printf(logPrefix+"failed to write resources to %s, error: %v", artPath, err)
		return err
	}

	// TODO: Move this verification into a unit test.
	// Verify that the .art file saved successfully.
	publishedBytes, err := ioutil.ReadFile(artPath)
	if err != nil {
		log.Printf(logPrefix+"failed to read resources from %s, error: %v", artPath, err)
		return err
	}
	if bytes.Compare(publishedBytes, publication.SerializedArtResources) != 0 {
		log.Fatalf(logPrefix+"mismatched bytes, published at %s: %v, serialized: %v",
			artPath, publishedBytes, publication.SerializedArtResources)
		return fmt.Errorf("bytes on disk out of sync")
	}

	// Write the signed publication to the artist's [pubkey].pub file.
	marshaledPublication, err := proto.Marshal(publication)
	if err != nil {
		log.Printf(logPrefix+"failed to marshal %v, error: %v", publication, err)
		return err
	}

	pubPath := fileServer.publicationPath(publication.Artist)
	err = ioutil.WriteFile(pubPath, marshaledPublication, 0644)
	if err != nil {
		log.Printf(logPrefix+"failed to write publication to %s, error: %v", pubPath, err)
		return err
	}

	return nil
}

func (fileServer *FileServer) readPublication(artistID string, pubkey string, publicationPath string) (*art.ArtistPublication, error) {
	const logPrefix = "FileServer readPublication "

	publishedData, err := ioutil.ReadFile(publicationPath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile %s error: %v", publicationPath, err)
		return nil, err
	}

	publication := art.ArtistPublication{}
	err = proto.Unmarshal(publishedData, &publication)
	if err != nil {
		log.Printf(logPrefix+"failed to unmarshal publication, error: %v", err)
		return nil, err
	}

	return &publication, err
}

func (fileServer *FileServer) readArtFile(artistID string, artFilePath string) error {
	const logPrefix = "FileServer readArtFile "

	artData, err := ioutil.ReadFile(artFilePath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile %s, error: %v", artFilePath, err)
		return err
	}

	resources := art.ArtResources{}
	err = proto.Unmarshal(artData, &resources)
	if err != nil {
		return err
	}

	return fileServer.indexResources(&resources)
}

func (fileServer *FileServer) Artists() (map[string]*art.Artist, error) {
	return fileServer.artists, nil
}

func (fileServer *FileServer) Artist(artistID string) (*art.Artist, error) {
	artist := fileServer.artists[artistID]
	if artist == nil {
		return nil, ErrArtNotFound
	}
	return artist, nil
}

func (fileServer *FileServer) Albums(artistId string) (map[string]*art.Album, error) {
	albums, found := fileServer.albums[artistId]
	if !found {
		// Read album info from file system.
		fmt.Errorf("not implemented")
	}

	return albums, nil
}

func (fileServer *FileServer) Tracks(artistID string) (map[string]*art.Track, error) {
	return fileServer.tracks[artistID], nil
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

// StorePublication saves a file with the published artist details, albums, tracks, and peers.
func (fileServer *FileServer) StorePublication(publication *art.ArtistPublication) error {
	const logPrefix = "fileServer storeToFileSystem "

	artistId := publication.Artist.ArtistId
	previouslyPublishedArtist := fileServer.artists[artistId]
	if previouslyPublishedArtist != nil &&
		previouslyPublishedArtist.Pubkey != publication.Artist.Pubkey &&
		previouslyPublishedArtist.Pubkey != "" {
		log.Printf(logPrefix+"update pubkey for /%s from %s to %s",
			artistId, previouslyPublishedArtist.Pubkey, publication.Artist.Pubkey)
		// TODO: validate that it's safe to replace
	}
	fileServer.artists[artistId] = publication.Artist

	// Read the resources from the publication.
	publishedResources, err := read(publication)
	if err != nil {
		log.Fatalf(logPrefix+"failed to read publication %v, error: %v", publication, err)
		return err
	}

	err = fileServer.savePublishedResources(publication, publishedResources)
	if err != nil {
		return err
	}

	err = fileServer.indexResources(publishedResources)

	return err
}

// indexResources indexes the resources for fast retrieval
func (fileServer *FileServer) indexResources(resources *art.ArtResources) error {
	const logPrefix = "FileServer indexResources "

	for _, artist := range resources.Artists {
		fileServer.artists[artist.ArtistId] = artist
	}
	for _, album := range resources.Albums {
		artistAlbums := fileServer.albums[album.ArtistId]
		if artistAlbums == nil {
			artistAlbums = make(map[string]*art.Album)
			fileServer.albums[album.ArtistId] = artistAlbums
		}
		artistAlbums[album.ArtistAlbumId] = album
	}
	for _, track := range resources.Tracks {
		artistTracks := fileServer.tracks[track.ArtistId]
		if artistTracks == nil {
			artistTracks = make(map[string]*art.Track)
			fileServer.tracks[track.ArtistId] = artistTracks
		}
		artistTracks[track.ArtistTrackId] = track
	}

	for _, peer := range resources.Peers {
		fileServer.peers[peer.Pubkey] = peer
	}

	return nil
}

// StoreArtist validates the given artist and stores it in memory.
func (fileServer *FileServer) StoreArtist(artist *art.Artist) error {
	const logPrefix = "FileServer StoreArtist "

	// validate the artist
	if artist.Pubkey == "" {
		log.Printf("FileServer StoreArtist reject artist missing Pubkey: %v", *artist)
		return fmt.Errorf("Failed to store artist missing Pubkey")
	}
	
	fileServer.artists[artist.ArtistId] = artist

	return nil
}

func (fileServer *FileServer) StoreAlbum(album *art.Album, publisher Publisher) error {
	const logPrefix = "FileServer StoreAlbum "

	publishingArtist, err := publisher.Artist()
	if err != nil {
		log.Fatalf(logPrefix+"failed to get Artist for publisher %v, error: %v", publisher, err)
		return err
	}

	albumArtist, err := fileServer.Artist(album.ArtistId)
	if err != nil {
		log.Fatalf(logPrefix+"failed to get artist %s for album %v, error: %v",
			album.ArtistId, album, err)
		return err
	}

	if publishingArtist.Pubkey != albumArtist.Pubkey {
		log.Printf(logPrefix+"skip StoreAlbum %v because publishing pubkey %v does not match album artist pubkey %s, error: %v",
			album, publishingArtist.Pubkey, albumArtist.Pubkey, err)
		return err
	}

	if album.ArtistId == "" || album.ArtistAlbumId == "" {
		log.Fatalf(logPrefix+"malformed album %v", album)
	}

	artistAlbums := fileServer.albums[album.ArtistId]
	if artistAlbums == nil {
		artistAlbums = make(map[string]*art.Album)
		fileServer.albums[album.ArtistId] = artistAlbums
	}

	artistAlbums[album.ArtistAlbumId] = album
	log.Printf(logPrefix+"stored album %v for publishing artist %v", album, publishingArtist)

	return nil
}

// StorePeer stores the peer in the in-memory database
// and saves it to the publisher's artist directory file system.
func (fileServer *FileServer) StorePeer(peer *art.Peer, publisher Publisher) error {
	const logPrefix = "FileServer StorePeer "

	publishingArtist, err := publisher.Artist()
	if err != nil {
		log.Fatalf(logPrefix+"failed to get Artist for publisher %v, error: %v", publisher, err)
		return err
	}

	log.Printf("FileServer StorePeer %v for publishing artist %v", peer, publishingArtist)
	if publishingArtist.Pubkey == peer.Pubkey {
		fileServer.peers[peer.Pubkey] = peer
	} else {
		log.Printf(logPrefix+"skip StorePeer %v because pubkey does not match artist %v, error: %v",
			peer, publishingArtist, err)
		return err
	}
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

// StoreTrack stores track in the in-memory database
// and eventually "publishes" (flushes) all resources to the file system.
func (fileServer *FileServer) StoreTrack(track *art.Track, publisher Publisher) error {
	const logPrefix = "FileServer StoreTrack "

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

func (fileServer *FileServer) StoreTrackPayload(track *art.Track, payload []byte) error {
	const logPrefix = "FileServer StoreTrackPayload "

	filename := fileServer.mp3Filename(track)
	containerDirectory := filepath.Dir(filename)

	err := os.MkdirAll(containerDirectory, 0755)
	if err != nil {
		log.Printf(logPrefix+"Failed to make directory %s, error: %v", containerDirectory, err)
		return err
	}

	err = ioutil.WriteFile(filename, payload, 0644)
	return err
}

func (fileServer *FileServer) Track(artistID string, trackID string) (*art.Track, error) {
	return fileServer.tracks[artistID][trackID], nil
}

func (fileServer *FileServer) TrackFilePath(track *art.Track) string {
	return fileServer.mp3Filename(track)
}

func (fileServer *FileServer) artPath(artist *art.Artist) string {
	return filepath.Join(fileServer.rootPath, artist.ArtistId, ".art")
}

func (fileServer *FileServer) publicationPath(artist *art.Artist) string {
	// TODO: validate params.
	if artist.ArtistId == "" {
		log.Fatalf("FileServer publicationPath failed for missing artist id in %v", artist)
	}
	if artist.Pubkey == "" {
		log.Fatalf("FileServer publicationPath no known pubkey for artist %s", artist.ArtistId)
	}
	return filepath.Join(fileServer.rootPath, artist.ArtistId, artist.Pubkey+".pub")
}

func (fileServer *FileServer) mp3Filename(track *art.Track) (filename string) {
	// TODO: sanitize filepath so peer cannot write outside the base path dir sandbox.
	return filepath.Join(fileServer.rootPath, track.ArtistId, track.ArtistTrackId+".mp3")
}
