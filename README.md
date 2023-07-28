# Nostrocket Engine
Problem: can't reach consensus about the current state of Nostrocket and Subrockets

This repo doesn't have an issue tracker because we are using Nostrocket to track problems. Please view the problem tracker using [Spacemen](nostrocket.github.io/spaceman)

### Nostrocket Engine is a replicated state machine built with Nostr and Bitcoin.

It's essentially a "Layer 2" which parses nostr events, and updates state if the event complies with the protocol.

The distributed consensus layer makes state immutable by putting a merkle root into the Bitcoin chain.

#### Replay Protection
`["r", "<last successful state-change event from this pubkey>"]`   
Events which do not have replay protection MUST NOT cause a state change.

#### Bitcoin 
`["btc", "<height int decimal>", "<header 32 bytes hex>", "<unix timestamp int decimal>"]`