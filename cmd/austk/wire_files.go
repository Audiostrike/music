// +build wireinject
// Wire builds wire_gen.go from
// 1. the signature from func injectArtServer (the injector function) in this stub file,
// 2. the expressions that injectArtServer passes to `wire.Build()`, and
// 3. the functions other than the injector.
// The `+build` directive above omits this file from released builds, which use generated wire_gen.go.

package main

import (
	audiostrike "github.com/audiostrike/music/internal"
	"github.com/google/wire"
)

func injectFileServer(artDirPath string) (s audiostrike.ArtServer, err error) {
	wire.Build(audiostrike.NewFileServer, useFileServer)
	return
}

func useFileServer(fileServer *audiostrike.FileServer) audiostrike.ArtServer {
	return fileServer
}
