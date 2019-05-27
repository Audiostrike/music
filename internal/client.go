package audiostrike

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	art "github.com/audiostrike/music/pkg/art"
	"github.com/cretz/bine/tor"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

type Client struct {
	artClient        art.ArtClient
	peerAddress      string
	torClient        *http.Client
	torProxy         string
	connectionCtx    context.Context
	connectionCancel context.CancelFunc
}

func NewClient(torProxy string, peerAddress string) (*Client, error) {
	const logPrefix = "client NewClient "
	ctx := context.Background()
	// Wait a few minutes to connect to tor network.
	connectionCtx, connectionCancel := context.WithTimeout(ctx, 3*time.Minute)
	artClient, err := newArtClient(connectionCtx, torProxy, peerAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"newArtClient error %v\n", err)
		return nil, err
	}
	torClient, err := NewTorClient(torProxy)
	if err != nil {
		return nil, err
	}
	client := &Client{
		torProxy:         torProxy,
		peerAddress:      peerAddress,
		artClient:        artClient,
		connectionCtx:    connectionCtx,
		connectionCancel: connectionCancel,
		torClient:        torClient,
	}
	return client, nil
}

func (client *Client) CloseConnection() {
	client.connectionCancel()
}

func newArtClient(ctx context.Context, torProxy string, endpoint string) (artClient art.ArtClient, err error) {
	const logPrefix = "client newArtClient "
	fmt.Println(logPrefix + "artClient ok")
	return
}

func (client *Client) GetAllArtByTor() (*art.ArtReply, error) {
	const logPrefix = "client GetAllArt "
	fmt.Printf(logPrefix+"with torClient proxy %v to http://%v\n", client.torProxy, client.peerAddress)
	response, err := client.torClient.Get("http://" + client.peerAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"torClient.get %v, error: %v\n", client.peerAddress, err)
		return nil, err
	}
	defer response.Body.Close()
	replyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"ReadAll response.Body error: %v\n", err)
		return nil, err
	}
	fmt.Printf(logPrefix+"Read reply: %v\n", string(replyBytes))
	var reply art.ArtReply
	err = proto.Unmarshal(replyBytes, &reply)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"Unmarshal reply error: %v\n", err)
		return nil, err
	}
	return &reply, nil
}

func (client *Client) GetArtByTor(artistId string, artistTrackId string) ([]byte, error) {
	const logPrefix = "client GetArtByTor "
	trackUrl := fmt.Sprintf(
		"http://%s/art/%s/%s",
		client.peerAddress, artistId, artistTrackId)
	response, err := client.torClient.Get(trackUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"torClient.get %v, error: %v\n", trackUrl, err)
		return nil, err
	}
	defer response.Body.Close()
	replyBytes, err := ioutil.ReadAll(response.Body)
	fmt.Printf(logPrefix+"Read reply: %v\n", string(replyBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"ReadAll response.Body error: %v\n", err)
		return nil, err
	}
	return replyBytes, nil
}

func (client *Client) GetAllArtByGrpc() (*art.ArtReply, error) {
	const logPrefix = "client GetAllArtByGrpc "
	torClient, err := tor.Start(client.connectionCtx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"tor.Start error: %v\n", err)
		return nil, err
	}
	defer torClient.Close()
	dialConf := tor.DialConf{
		ProxyAddress: "localhost:9050",
	}
	dialer, err := torClient.Dialer(client.connectionCtx, &dialConf)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"tor.Dialer error: %v\n", err)
		return nil, err
	}

	artRequest := art.ArtRequest{}
	fmt.Printf(logPrefix+"Dial peer %v by over tor...\n", client.peerAddress)
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
		fmt.Fprintf(os.Stderr, logPrefix+"Dial peer error: %v\n", err)
		return nil, err
	}
	defer peerConnection.Close()

	fmt.Printf(logPrefix+"GetArt from peer %v...\n", client.peerAddress)
	artReply, err := client.artClient.GetArt(client.connectionCtx, &artRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"artClient.GetArt error: %v\n", err)
		return nil, err
	}
	return artReply, nil
}

func NewTorClient(torProxy string) (*http.Client, error) {
	const logPrefix = "client NetTorClient "
	torProxyUrl, err := url.Parse(torProxy)
	if err != nil {
		fmt.Fprintf(os.Stderr, logPrefix+"url.Parse %v error: %v\n", torProxy, err)
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(torProxyUrl),
		},
	}, nil
}
