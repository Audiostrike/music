package audiostrike

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"

	art "github.com/audiostrike/music/pkg/art"
	"github.com/golang/protobuf/proto"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
	"strings"
)

type LightningPublisher struct {
	lightningClient  lnrpc.LightningClient
	publishingArtist *art.Artist
}

func NewLightningPublisher(cfg *Config, publishingArtist *art.Artist) (*LightningPublisher, error) {
	const logPrefix = "lightningPublisher NewLightningPublisher "

	// Get the TLS credentials for the lnd server.
	tlsCertFilePath, err := tlsCertPath(cfg)
	if err != nil {
		log.Fatalf(logPrefix+"failed to get tls cert path, error: %v", err)
		return nil, err
	}
	// The second paramater here is serverNameOverride, set to ""
	// except to override the virtual host name of authority in test requests.
	lndTlsCreds, err := credentials.NewClientTLSFromFile(tlsCertFilePath, "")
	if err != nil {
		log.Fatalf(logPrefix+"failed to get tls credentials from %s, error: %v",
			tlsCertFilePath, err)
		return nil, err
	}

	lndMacaroon, err := macaroonFromFile(cfg)
	if err != nil {
		log.Printf(logPrefix+"UnmarchalBinary macaroon error: %v\n", err)
		return nil, err
	}

	lndOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(lndTlsCreds),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(lndMacaroon)),
	}

	lndGrpcEndpoint := fmt.Sprintf("%v:%d", cfg.LndHost, cfg.LndGrpcPort)
	log.Printf(logPrefix+"Dial lnd grpc at %v...", lndGrpcEndpoint)
	lndConn, err := grpc.Dial(lndGrpcEndpoint, lndOpts...)
	if err != nil {
		log.Printf(logPrefix+"Dial lnd error: %v", err)
		return nil, err
	}
	lndClient := lnrpc.NewLightningClient(lndConn)

	// Set the publishing Artist for this lightningPublisher with the configured ArtistID and Name.
	return &LightningPublisher{
		lightningClient:  lndClient,
		publishingArtist: publishingArtist,
	}, nil
}

func (lightningPublisher *LightningPublisher) Artist() (*art.Artist, error) {
	return lightningPublisher.publishingArtist, nil
}

// getAllArtHandler handles a request to get all the art from the ArtService.
func (lightningPublisher *LightningPublisher) Publish(resources *art.ArtResources) (*art.ArtistPublication, error) {
	const logPrefix = "(*LightningPublisher) Publish "

	ctx := context.Background()
	marshaledResources, err := proto.Marshal(resources)
	if err != nil {
		log.Printf(logPrefix+"Marshal %v, error: %v", resources, err)
		return nil, err
	}
	signMessageInput := lnrpc.SignMessageRequest{Msg: marshaledResources}
	signMessageResult, err := lightningPublisher.lightningClient.SignMessage(ctx, &signMessageInput)
	if err != nil {
		log.Printf(logPrefix+"SignMessage error: %v", err)
		return nil, err
	}
	publicationSignature := signMessageResult.Signature
	log.Printf(logPrefix+"Signed message %v, signature: %v", resources, publicationSignature)

	return &art.ArtistPublication{
		Artist:                 lightningPublisher.publishingArtist,
		Signature:              publicationSignature,
		SerializedArtResources: marshaledResources,
	}, nil
}

func (lightningPublisher *LightningPublisher) ValidatePublication(publication *art.ArtistPublication) (*art.ArtResources, error) {
	const logPrefix = "lightningPublisher ValidatePublication "

	ctx := context.Background()
	verifyMessageRequest := lnrpc.VerifyMessageRequest{
		Msg:       publication.SerializedArtResources,
		Signature: publication.Signature,
	}
	verifyMessageResponse, err := lightningPublisher.lightningClient.VerifyMessage(ctx, &verifyMessageRequest)
	if err != nil {
		log.Printf(logPrefix+"failed to verify message, error: %v", err)
		return nil, err
	}
	if !verifyMessageResponse.Valid {
		log.Printf(logPrefix+"Signature %s is not valid for message %v", publication.Signature, publication.SerializedArtResources)
		return nil, fmt.Errorf("Signature failed verification")
	}
	if verifyMessageResponse.Pubkey != publication.Artist.Pubkey {
		log.Printf(logPrefix+"Signature pubkey %s does not match pubkey for publishing artist %v",
			verifyMessageResponse.Pubkey, publication.Artist)
		return nil, err
	}

	artResources := art.ArtResources{}
	err = proto.Unmarshal(publication.SerializedArtResources, &artResources)
	if err != nil {
		log.Printf(logPrefix+"Unmarshal error: %v", err)
		return nil, err
	}
	return &artResources, nil
}

// Pubkey returns the pubkey for the lnd server,
// which clients can use to authenticate publications from this node.
func (lightningPublisher *LightningPublisher) Pubkey() (Pubkey, error) {
	return pubkey(lightningPublisher.lightningClient)
}

func pubkey(lightningClient lnrpc.LightningClient) (Pubkey, error) {
	ctx := context.Background()
	getInfoRequest := lnrpc.GetInfoRequest{}
	getInfoResponse, err := lightningClient.GetInfo(ctx, &getInfoRequest)
	if err != nil {
		return "", err
	}
	pubkey := Pubkey(getInfoResponse.IdentityPubkey)

	return pubkey, nil
}

func (lightningPublisher *LightningPublisher) NewInvoice(tracks []*art.Track, amount int32, amountUnits art.Bolt11AmountMultiplier) (*art.Invoice, error) {
	const logPrefix = "(*LightningPublisher) Invoice "

	ctx := context.Background()
	memo, err := invoiceMemo(tracks)
	if err != nil {
		log.Printf(logPrefix+"Failed to generate memo for invoice, error: %v", err)
		return nil, err
	}
	valueSatoshis, err := valueSatoshis(amount, amountUnits)
	if err != nil {
		log.Printf(logPrefix+"failed to convert amount %d %s to satoshis, error: %v",
			amount, amountUnits.String(), err)
		return nil, err
	}
	lightningInvoice := lnrpc.Invoice{
		Memo:  memo,
		Value: valueSatoshis,
	}
	addInvoiceResponse, err := lightningPublisher.lightningClient.AddInvoice(ctx, &lightningInvoice)
	if err != nil {
		log.Printf(logPrefix+"Failed to add invoice: %v, error: %v", lightningInvoice, err)
		return nil, err
	}

	artist, err := lightningPublisher.Artist()
	if err != nil {
		log.Printf(logPrefix+"failed to get artist to issue invoice, error: %v", err)
		return nil, err
	}
	
	return &art.Invoice{
		ArtistId: artist.ArtistId,
		// BOLT-11 encoded lightning invoice, e.g. "lnbc2u1..." for a 2µBTC invoice
		Bolt11PaymentRequest:   addInvoiceResponse.PaymentRequest,
		LightningPaymentHash:   addInvoiceResponse.RHash[:],
		Tracks:                 tracks,
		Bolt11Amount:           amount,
		Bolt11AmountMultiplier: amountUnits,
	}, nil
}

func invoiceMemo(tracks []*art.Track) (string, error) {
	if len(tracks) == 0 {
		return "", fmt.Errorf("no tracks for invoice")
	}
	// TODO: build this memo up to 1023 bytes. Any greater amount would need a memo hash.
	ids := make([]string, 0, len(tracks))
	for _, track := range tracks {
		if len(track.Presentations) == 0 {
			ids = append(ids, track.ArtistId+"/"+track.ArtistTrackId)
			continue
		}
		for _, presentation := range track.Presentations {
			if len(presentation.Segments) == 0 {
				ids = append(ids, presentation.FullId)
				continue
			}
			for _, segment := range presentation.Segments {
				ids = append(ids, segment.FullId)
			}
		}
	}
	memo := strings.Join(ids, " ")
	return memo, nil
}

func valueSatoshis(amount int32, amountUnits art.Bolt11AmountMultiplier) (int64, error) {
	switch amountUnits {
	case art.Bolt11AmountMultiplier_BITCOIN_BIT:
		return int64(amount) * 100, nil
	}
	return 0, fmt.Errorf("valueSatoshis not implemented")
}

func (lightningPublisher *LightningPublisher) Invoice(paymentHash *lntypes.Hash) (*art.Invoice, error) {
	return nil, fmt.Errorf("LightningPublisher Invoice retrieval not implemented")
}

// tlsCertPath gets the TlsCertPath from the given Config.
// If TlsCertPath is "" (not configured), this defaults to the user's ~/.lnd/tls.cert file.
func tlsCertPath(cfg *Config) (string, error) {
	if cfg.TlsCertPath == "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", err
		}
		tlsCertFilePath := currentUser.HomeDir + "/.lnd/tls.cert"
		return tlsCertFilePath, nil
	}
	return cfg.TlsCertPath, nil
}

// macaroonFromFile gets a Macaroon with the contents of the configured or default lnd macaroon.
// The default is the Macaroon in the user's ~/.lnd/data/chain/bitcoin/regtest/admin.macaroon file.
func macaroonFromFile(cfg *Config) (*macaroon.Macaroon, error) {
	const logPrefix = "lightningnode macaroonFromFile "

	// Get the macaroon for lnd grpc requests.
	// This macaroon must support creating invoices and signing messages.
	macaroonFilePath, err := macaroonPath(cfg)
	if err != nil {
		log.Fatalf(logPrefix+"failed to get macaroon from config %v, error: %v",
			cfg, err)
		return nil, err
	}
	macaroonData, err := ioutil.ReadFile(macaroonFilePath)
	if err != nil {
		log.Printf(logPrefix+"ReadFile %s, error: %v", cfg.MacaroonPath, err)
		return nil, err
	}

	lndMacaroon := macaroon.Macaroon{}
	err = lndMacaroon.UnmarshalBinary(macaroonData)
	if err != nil {
		log.Printf(logPrefix+"UnmarshalBinary macaroon error: %v", err)
		return nil, err
	}
	return &lndMacaroon, nil
}

// macaroonPath gets the MacaroonPath from the given Config.
// If MacaroonPath is "" (not configured), this defaults to the user's ~/.lnd admin macaroon
// for a local bitcoin regtest network so devs/testers can mine their own blocks to pay with free coins.
func macaroonPath(cfg *Config) (string, error) {
	if cfg.MacaroonPath == "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", err
		}
		// Hardcode network to regtest for now
		// to avoid risking real funds and to avoid relying on testnet miners/bandwidth.
		// Later this should become a configurable parameter defaulting to testnet.
		// Default to mainnet only in production releases.
		network := "regtest"
		macaroonPath := currentUser.HomeDir + "/.lnd/data/chain/bitcoin/" + network + "/admin.macaroon"
		return macaroonPath, nil
	}
	return cfg.MacaroonPath, nil
}
