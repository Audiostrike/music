# Audiostrike

Audiostrike is an open-source initiative to let artists create a free market for their listeners to buy and stream their music. The music artists themselves will publish, own, and control the decentralized music directory that listeners can access with a music player to buy and stream the atrists' music.

This git repo is meant to define the protocol and to implement a prototype server for music publication, discovery, purchase, and streaming.

## Decentralized Music Directory

An open, decentralized music directory will be built by music artists themselves when they publish, price, and host their own music on an Audiostrike node (an `lnd` node with a feature bit and some REST/grpc services). Listeners will discover, purchase, and stream from the directory with music players implemented in related projects.

## Open-Source Music Server

To use the Audiostrike protocol, artists will upload mp3 files of their own music to their own Audiostrike nodes and will configure pricing and streaming options (streaming bit rate options, price per segment at given bit rates, duration of any "free" or nearly free intro segment for fans to sample before buying the remainder of the track, etc.). The Audiostrike node will convert the mp3 files into DASH-compatible media descriptors and DASH segments with the configured bit rates and will publish (possibly with one or more new types of gossip inventory message) the music metadata and streaming/purchase options. From there, other artists and fans will discover the published music to purchase and stream.

Artists will each generate a public/private key pair to sign each of their music publications. Other artists and fans can use such signatures and public keys to authenticate the legitimate artist's ownership of the published music.

In a typical stream-purchasing scenario, a listener's music player will use a bitcoin micro-payment over the Lightning Network to purchase from the artist a secret that functions (a) as a DoS-mitigating access code for downloading purchased music segments and (b) optionally (configurable by each artist, not expected to be implemented in the initial prototype) as an MPEG-DASH Common Encryption (CENC) key for the purchased music segments. The listener's music player provides the access code to the artist's music server to download the media presentation description and purchased segments for streaming. Music players will typically (given storage constraints) save streamed segments for repeated playback, especially useful when the music player is offline or low on funds. Other than the download-access code and the optional CENC key, no DRM or other types of content protection are expected to be implemented in the Audiostrike protocol.

## Artist Authentication

The decentralized music directory should help listeners confidently buy music from the artists who create and own it rather than unwittingly buying from someone fraudulently posing as the artist. This protocol has no central system of artist authentication, so each listener should individually authenticate at least one artist known by the listener as the legitimate creator or owner of music published in the directory. *(This verification could potentially be deferred while the listener temporarily trusts a seed directory, but direct listener-driven authentication is critical to keeping the directory open and decentralized.)*

To let anyone authenticate an artist as the legitimate owner of an Audiostrike music publication, each artist will use his or her own Audiostrike node to create a public and private key for publishing music. The artist should communicate that public key to fans, typically by publishing the public key to an Internet resource that fans know the artist controls. For example, Alice in Chains may publish their Audiostrike music-publishing public key to https://aliceinchains.com/pubkey or to a subresource of https://www.facebook.com/aliceinchains or https://bandcamp.com/tag/alice-in-chains and link to that resource from their Audiostrike publication. Any fan of Alice in Chains could then verify that the signature for music publications purporting to be from Alice in Chains matches the public key that Alice in Chains published to their Internet resource. *(The method of direct verification of artists' public keys is not yet fleshed out but may eventually include integration between the listener's music player and mobile apps such as Bandcamp, Facebook, Twitter, etc.)*

To facilitate artist authentication by listeners, a tree of artist-authentication attestations will be exposed by the Audiostrike protocol by composing attestations from individual artists who (a) verify/authenticate other artists' public keys and (b) sign and publish such authentication attestations along with their own published music in the directory. For example, Alice in Chains may verify that a given public key belongs to B.o.B. through any trusted communication with Bobby Ray Simmons, Jr., perhaps by in-person conversation or by checking an Internet resource he controls. Alice in Chains may then publish to their own section of the music directory an attestation to the authentic ownership of that public key by B.o.B. along with a URL to B.o.B.'s publications in the open music directory. Similarly, B.o.B. may verify/authenticate/attest to the public key for music artist Charles Hamilton. By doing so, a fan of Alice in Chains who only authenticates their one public key can automatically get a larger tree of artist authentication by proxy, letting the music player use Alice in Chains' attestation of B.o.B.'s public key to authenticate music published by B.o.B. and B.o.B.'s attestation of Charles Hamilton's public key to authenticate music published by Charles Hamilton. *(It is not yet clear how best to consolidate individual artists' publications in the protocol such that each music player need not query a large portion of the network of music servers. Presumably the Audiostrike servers will gossip artists' publications while minimizing redundancy with something like https://github.com/sipa/minisketch)*

Conflicts in attested keys are expected to arise in an open decentralized directory like this, e.g. two different public keys may be attested for the band Depeche Mode. Listeners need a means of resolving such conflicts in order to avoid paying a fraudster instead of the legitimate artist. Additional design is needed to help listeners easily identify which of conflicting attestations of artist's public keys to trust. One potential tactic to minimize the listener's burden for conflict resolution is to default to the identity of the attestation with the fewest number of "hops" from the listener's authentication "roots" (public keys directly authenticated by the listener). Another is simply to alert the listener to the conflict and prompt a choice while showing the attestation path with the fewest hops between each conflicting attestation and the artist public keys directly authenticated by the listener. *(It is not yet clear whether it would be helpful or spammy/DoSy to broadcast details of such manual conflict resolution by listeners when they directly authenticate a public key by checking an Internet resource known to be controlled by the given artist.)*

## Music players

Music players can use the Audiostrike protocol to browse the open, decentralized music directory to find music that matches the listener's preferences, e.g. streaming budget per hour, genres, favorite influences, etc. Initially a reference client (an Android native music player, potentially followed by a cross-platform app based on React) will be built in this project but this protocol is expected to be incorporated eventually into existing music players.

To bootstrap discovery of artists' music publications, any given music player should include at least one directory-seed URL pointing to a batch of artists' music publications to seed discovery. For example, the reference client will include a directory-seed URL of https://audiostrike.net and that resource will include links to any artist who publishes to it on a first-come, first-served basis. Additional design may be needed to ensure that this apparent point of centralization for the prototype player does not become a contentious or abused resource.

## Free market for music

With Audiostrike, artists can set their own prices, publish their own music, and stream it to their fans. Listeners can set their own music budget, discover music, and buy it to stream directly from the artists they love. This removes the middlemen (Apple, Google, Pandora, Spotify, etc. as well as banks and payment processors) who can no longer dictate prices, take a unilateral cut of the revenue, or censor anyone. Streaming rates will be determined through catallaxy by artists and fans, perhaps beginning somewhere around one bitcoin "bit" (a millionth of a bitcoin, about half a cent in US Dollars as of April 2019) per hour of streaming. Listeners could configure their music player to buy only music priced low enough to hit that hourly target when combined with zero-cost repetitions of music already purchased/stored or they may increase or decrease their budget to explore premium music or to save money. Artists could choose to get additional exposure by reducing streaming prices below the average rates or they could raise the price of their own music if they think the public knows them well enough and demands their music urgently enough to be worth a premium price. Artists and fans will have full control over their own participation in this new free market for music.

## Other Open-Source Protocols and Initiatives for Music Artists

The Open Music Initiative is an open-source protocol for identifying music creators and holders of music rights. Its mission is compatible with but orthogonal to Audiostrike. The reference client in this project may eventually use the protocol defined by the Open Music Initiative to help verify that the legitimate creators and rights holders are the ones paid for music published and consumed through Audiostrike.

Bandcamp or similar services may eventually use the Audiostrike protocol to enable bitcoin-based payments, especially for any artist who cannot or does not wish to host his or her own music server. Such artists could let a service like Bandcamp to host their music server, which would require a custodial wallet to receive the payments. More design work may be needed in this area to help these custodians adopt the protocol without controlling the artist's publishing key.

Open-source music players such as https://github.com/kabouzeid/Phonograph , https://github.com/naman14/TimberX , and others may eventually incorporate the Audiostrike protocol to give their users access to the new free market that artists will build and own.
