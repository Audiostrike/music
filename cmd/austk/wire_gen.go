// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package main

import (
	"github.com/audiostrike/music/internal"
)

// Injectors from wire_db.go:

func injectDbServer(db *audiostrike.AustkDb, artRootPath string) audiostrike.ArtServer {
	dbServer := audiostrike.NewDbServer(db, artRootPath)
	artServer := useDbServer(dbServer)
	return artServer
}

// Injectors from wire_files.go:

func injectFileServer(artDirPath string) (audiostrike.ArtServer, error) {
	fileServer, err := audiostrike.NewFileServer(artDirPath)
	if err != nil {
		return nil, err
	}
	artServer := useFileServer(fileServer)
	return artServer, nil
}

// Injectors from wire_lightning.go:

func injectLightningNode(cfg *audiostrike.Config, artServer audiostrike.ArtServer) (audiostrike.Publisher, error) {
	lightningNode, err := audiostrike.NewLightningNode(cfg, artServer)
	if err != nil {
		return nil, err
	}
	publisher := useLightningNode(lightningNode)
	return publisher, nil
}

// Injectors from wire_publisher.go:

func injectPublisher(cfg *audiostrike.Config, artServer audiostrike.ArtServer, publisher audiostrike.Publisher) (*audiostrike.AustkServer, error) {
	austkServer, err := audiostrike.NewAustkServer(cfg, artServer, publisher)
	if err != nil {
		return nil, err
	}
	return austkServer, nil
}

// wire_db.go:

func useDbServer(dbServer *audiostrike.DbServer) audiostrike.ArtServer {
	return dbServer
}

// wire_files.go:

func useFileServer(fileServer *audiostrike.FileServer) audiostrike.ArtServer {
	return fileServer
}

// wire_lightning.go:

func useLightningNode(lightningNode *audiostrike.LightningNode) audiostrike.Publisher {
	return lightningNode
}
