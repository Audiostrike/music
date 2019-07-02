package audiostrike

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	faifacemp3 "github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	mikkyangid3 "github.com/mikkyang/id3-go"
	"io/ioutil"
	"log"
	"strings"
)

// Mp3 exposes the Tags (mp3 metadata) and bytes of a given .mp3 file.
type Mp3 struct {
	file             *os.File
	buffer           []byte
	length           int
	position         int
	Tags             map[string]string
	playbackFinished chan bool
}

// OpenMp3ToRead opens an mp3 file to read its data and tags (metadata)
func OpenMp3ToRead(fileName string) (mp3 *Mp3, err error) {
	// Read the file.
	var file *os.File
	file, err = os.Open(fileName)
	if err != nil {
		return
	}

	// Read the mp3 tags.
	var id3File *mikkyangid3.File
	id3File, err = mikkyangid3.OpenForRead(fileName)
	if err != nil {
		return
	}
	defer id3File.Close()
	var tags map[string]string
	tags, err = parseTags(id3File)
	if err != nil {
		return
	}

	// Return the Mp3 struct with the file and mp3 tags.
	mp3 = &Mp3{
		file: file,
		Tags: tags,
	}
	return
}

func OpenMp3ForTrackToRead(artistId string, artistTrackId string) (mp3 *Mp3, err error) {
	fileName := buildFileName(artistId, artistTrackId)
	mp3, err = OpenMp3ToRead(fileName)
	return
}

// SaveForTrack creates a file in the canonical location for artistId and artistTrackId
// and writes into it the bytes from the previously opened .mp3 file.
// An error is returned if an .mp3 file was not already opened.
func (mp3 *Mp3) SaveForTrack(artistId string, artistTrackId string) error {
	const logPrefix = "mp3 SaveForTrack "

	if mp3.file == nil {
		return fmt.Errorf("mp3 SaveForTrack no opened file to save for %s/%s", artistId, artistTrackId)
	}

	destinationFilename := buildFileName(artistId, artistTrackId)
	err := mkDirectoriesForTrack(artistId, artistTrackId)
	if err != nil {
		log.Printf(logPrefix+"MkDirectoriesForTrack %s/%s, error: %v", artistId, artistTrackId, err)
		return err
	}

	destinationFile, err := os.Create(destinationFilename)
	if err != nil {
		log.Printf(logPrefix+"os.Create %s, error: %v", destinationFilename, err)
		return err
	}

	mp3Bytes, err := mp3.ReadBytes()
	if err != nil {
		log.Printf(logPrefix+"mp3.ReadBytes, error: %v", err)
		return err
	}

	_, err = destinationFile.Write(mp3Bytes)
	return err
}

func WriteTrack(artistId string, artistTrackId string, bytes []byte) error {
	err := mkDirectoriesForTrack(artistId, artistTrackId)
	if err != nil {
		return err
	}

	filename := buildFileName(artistId, artistTrackId)
	err = ioutil.WriteFile(filename, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func buildFileName(artistId string, artistTrackId string) (filename string) {
	// TODO: make base path configurable, defaulting to ./tracks/
	// TODO: sanitize filepath so peer cannot write outside the base path dir sandbox.
	return fmt.Sprintf("./tracks/%s/%s.mp3", artistId, artistTrackId)
}

func mkDirectoriesForTrack(artistId string, artistTrackId string) error {
	const logPrefix = "mp3 mkDirectoriesForTrack "

	dirPath := fmt.Sprintf("./tracks")
	err := os.Mkdir(dirPath, 0777)
	if err != nil {
		// If directory already exists, swallow this error.
		log.Printf(logPrefix+"Mkdir ./tracks error: %v", err)
		// TODO: fail more loudly if a different type of error prevents saving tracks.
	}

	dirPath = fmt.Sprintf("./tracks/%s", artistId)
	err = os.Mkdir(dirPath, 0777)
	if err != nil {
		// If directory already exists, swallow this error.
		log.Printf(logPrefix+"Mkdir ./tracks/%s error: %v", artistId, err)
		// TODO: fail more loudly if a different type of error prevents saving tracks.
	}

	pathComponents := strings.Split(artistTrackId, "/")
	for _, pathComponent := range pathComponents[:len(pathComponents)-1] {
		dirPath = dirPath + "/" + pathComponent
		err = os.Mkdir(dirPath, 0777)
		if err != nil {
			log.Printf(logPrefix+"Mkdir %s error: %v", dirPath, err)
		}
	}

	return nil
}

func parseTags(file *mikkyangid3.File) (map[string]string, error) {
	tags := map[string]string{
		"Artist": file.Artist(),
		"Album":  file.Album(),
		"Title":  file.Title(),
	}
	return tags, nil
}

func (mp3 *Mp3) ArtistName() string {
	return mp3.Tags["Artist"]
}

func (mp3 *Mp3) AlbumTitle() (string, bool) {
	albumTitle := mp3.Tags["Album"]
	return albumTitle, albumTitle != ""
}

func (mp3 *Mp3) Title() string {
	return mp3.Tags["Title"]
}

// ReadBytes returns the raw data from the .mp3 file.
func (mp3 *Mp3) ReadBytes() ([]byte, error) {
	// If buffer already has the bytes, return them.
	if mp3.buffer != nil {
		return mp3.buffer, nil
	}

	// Otherwise read the bytes from the file into buffer.
	fileInfo, err := mp3.file.Stat()
	if err == nil {
		mp3.buffer = make([]byte, fileInfo.Size())
		_, err = mp3.file.Read(mp3.buffer)
	}
	return mp3.buffer, err
}

func (mp3 *Mp3) PlayAndWait() error {
	trackStreamer, format, err := faifacemp3.Decode(mp3.file)
	if err != nil {
		log.Printf("Failed to decode mp3, error: %v", err)
		return err
	}
	defer trackStreamer.Close()
	mp3.playbackFinished = make(chan bool)
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/5))
	speaker.Play(beep.Seq(trackStreamer, beep.Callback(func() {
		mp3.playbackFinished <- true
	})))

	<-mp3.playbackFinished

	return nil
}
