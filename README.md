# Nostrocket Engine
Problem: can't reach consensus about the current state of Nostrocket and Subrockets

### Nostrocket Engine is a replicated state machine built with Nostr and Bitcoin.

It's essentially a Bitcoin "Layer 2" which parses nostr events, updates state if the event complies with the protocol, and then publishes the new state as nostr event which can be consumed by nostr clients.

The distributed consensus layer makes state immutable by putting a merkle root into the Bitcoin chain.

