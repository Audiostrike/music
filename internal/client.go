package audiostrike

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	art "github.com/audiostrike/music/pkg/art"
	"github.com/cretz/bine/tor"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"log"
)

type Client struct {
	peerAddress      string
	torClient        *http.Client
	torProxy         string
	connectionCtx    context.Context
	connectionCancel context.CancelFunc
}

// NewClient creates a new austk Client to communicate over torProxy with peerAddress.
func NewClient(torProxy string, peerAddress string) (*Client, error) {
	const logPrefix = "client NewClient "

	ctx := context.Background()
	// Wait a few minutes to connect to tor network.
	connectionCtx, connectionCancel := context.WithTimeout(ctx, 3*time.Minute)

	torClient, err := newTorClient(torProxy)
	if err != nil {
		return nil, err
	}

	client := &Client{
		torProxy:         torProxy,
		peerAddress:      peerAddress,
		connectionCtx:    connectionCtx,
		connectionCancel: connectionCancel,
		torClient:        torClient,
	}
	return client, nil
}

// CloseConnection closes the onion-routing connection to the peer.
// This should be called after completing a session with a Client obtained by NewClient.
func (client *Client) CloseConnection() {
	client.connectionCancel()
}

// SyncFromPeer gets art-directory records (music metadata) from client's peer over tor and imports it into db.
func (client *Client) SyncFromPeer(db *AustkDb) ([]*art.Track, error) {
	const logPrefix = "client SyncFromPeer "

	reply, err := client.GetAllArtByTor()
	if err != nil {
		log.Fatalf(logPrefix+"GetAllArtByTor <-%v<-%v error: %v", client.torProxy, client.peerAddress, err)
	}

	err = client.importArtReply(reply, db)
	if err != nil {
		log.Fatalf(logPrefix+"importArtReply error: %v", err)
	}
	return reply.Tracks, err
}

func (client *Client) importArtReply(artReply *art.ArtReply, db *AustkDb) (err error) {
	const logPrefix = "austk importArtReply "
	var errors []error
	for _, artist := range artReply.Artists {
		err = db.PutArtist(artist)
		if err != nil {
			errors = append(errors, err)
		}
	}

	for _, track := range artReply.Tracks {
		err = db.PutTrack(track)
		if err != nil {
			errors = append(errors, err)
		}
	}

	for _, peer := range artReply.Peers {
		err = db.PutPeer(peer)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		log.Printf(logPrefix+"%v errors:", len(errors))
		for _, err = range errors {
			log.Printf(logPrefix+"\terror: %v", err)
		}
		return errors[0] // return the first error.
	}
	return nil
}

// DownloadTracks downloads tracks over tor from the peer whose pubkey matches the track artist.
//
// The .mp3 file is written as `./tracks/{ArtistId}/{ArtistTrackId}.mp3`
// That is, tracks download under an artist-specific subdirectory of ./tracks
// with filenames from the track's ArtistTrackId.
func (client *Client) DownloadTracks(tracks []*art.Track, db *AustkDb) (err error) {
	const logPrefix = "client DownloadTracks "
	var errors []error
	for _, track := range tracks {
		trackArtist, err := db.SelectArtist(track.ArtistId)
		if err != nil {
			errors = append(errors, err)
			continue // to next track
		}

		peer, err := db.SelectPeer(trackArtist.Pubkey)
		if peer == nil {
			err = fmt.Errorf("no known peer owns pubkey %s for %s/%s.mp3",
				trackArtist.Pubkey, track.ArtistId, track.ArtistTrackId)
			errors = append(errors, err)
			continue // to next track
		}

		replyBytes, err := client.GetTrackByTor(track.ArtistId, track.ArtistTrackId)
		if err != nil {
			errors = append(errors, err)
			continue // to next track
		}

		err = WriteTrack(track.ArtistId, track.ArtistTrackId, replyBytes)
		if err != nil {
			errors = append(errors, err)
			continue // to next track
		}
	}

	if len(errors) > 0 {
		log.Printf(logPrefix+"%v errors:", len(errors))
		for _, err = range errors {
			log.Printf(logPrefix+"\terror: %v", err)
		}
		return errors[0] // return the first error
	}
	
	return nil
}

// GetAllArtByTor gets the art-directory music metadata over tor from the client's peer.
func (client *Client) GetAllArtByTor() (*art.ArtReply, error) {
	const logPrefix = "client GetAllArtByTor "
	response, err := client.torClient.Get("http://" + client.peerAddress)
	if err != nil {
		log.Printf(logPrefix+"torClient.Get %v, error: %v", client.peerAddress, err)
		return nil, err
	}
	defer response.Body.Close()
	log.Printf(logPrefix+"torClient %v did Get http://%v", client.torProxy, client.peerAddress)

	// Read the reply into an ArtReply.
	replyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf(logPrefix+"ReadAll response.Body error: %v", err)
		return nil, err
	}
	var reply art.ArtReply
	err = proto.Unmarshal(replyBytes, &reply)
	if err != nil {
		log.Printf(logPrefix+"Unmarshal reply error: %v", err)
		return nil, err
	}
	
	return &reply, nil
}

// GetTrackByTor gets the track (mp3 bytes by http over tor) with artistTrackId by the artist with artistId.
func (client *Client) GetTrackByTor(artistId string, artistTrackId string) ([]byte, error) {
	const logPrefix = "client GetArtByTor "
	
	trackUrl := fmt.Sprintf("http://%s/art/%s/%s",
		client.peerAddress, artistId, artistTrackId)
	log.Printf(logPrefix+"Get %s...", trackUrl)
	response, err := client.torClient.Get(trackUrl)
	if err != nil {
		log.Printf(logPrefix+"torClient.get %v, error: %v", trackUrl, err)
		return nil, err
	}
	defer response.Body.Close()

	// Read the reply and return the bytes.
	replyBytes, err := ioutil.ReadAll(response.Body)
	log.Printf(logPrefix+"Read %d-byte reply", len(replyBytes))
	if err != nil {
		log.Printf(logPrefix+"ReadAll response.Body error: %v", err)
		return nil, err
	}
	return replyBytes, nil
}

// GetAllArtByGrpc is similar to GetAllArtByTor but uses Grpc rather than raw http over tor.
// This is dead code for now, as GetAllArtByTor seems to expose the needed functionality.
// This code may be revived if fields must be specified in the ArtRequest, e.g. for filtering results.
func (client *Client) GetAllArtByGrpc() (*art.ArtReply, error) {
	const logPrefix = "client GetAllArtByGrpc "
	torClient, err := tor.Start(client.connectionCtx, nil)
	if err != nil {
		log.Printf(logPrefix+"tor.Start error: %v", err)
		return nil, err
	}
	defer torClient.Close()
	dialConf := tor.DialConf{
		ProxyAddress: "localhost:9050",
	}
	dialer, err := torClient.Dialer(client.connectionCtx, &dialConf)
	if err != nil {
		log.Printf(logPrefix+"tor.Dialer error: %v", err)
		return nil, err
	}

	artRequest := art.ArtRequest{}
	log.Printf(logPrefix+"Dial peer %v by over tor...", client.peerAddress)
	peerConnection, err := grpc.DialContext(
		client.connectionCtx,
		client.peerAddress,
		grpc.FailOnNonTempDialError(true),
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithDialer(func(address string, timeout time.Duration) (net.Conn, error) {
			dialCtx, dialCancel := context.WithTimeout(client.connectionCtx, timeout)
			defer dialCancel()
			return dialer.DialContext(dialCtx, "tcp", address)
		}),
	)
	if err != nil {
		log.Printf(logPrefix+"Dial peer error: %v", err)
		return nil, err
	}
	defer peerConnection.Close()

	log.Printf(logPrefix+"GetArt from peer %v...", client.peerAddress)
	artClient := art.NewArtClient(peerConnection)
	artReply, err := artClient.GetArt(client.connectionCtx, &artRequest)
	if err != nil {
		log.Printf(logPrefix+"artClient.GetArt error: %v", err)
		return nil, err
	}
	return artReply, nil
}

func newTorClient(torProxy string) (*http.Client, error) {
	const logPrefix = "client NetTorClient "
	torProxyUrl, err := url.Parse(torProxy)
	if err != nil {
		log.Printf(logPrefix+"url.Parse %v error: %v", torProxy, err)
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(torProxyUrl),
		},
	}, nil
}
