# Nostrocket Engine
Problem: can't reach consensus about the current state of Nostrocket and Subrockets

### Nostrocket Engine is a replicated state machine built with Nostr and Bitcoin.

It's essentially a Bitcoin "Layer 2" which parses nostr events, updates state if the event complies with the protocol, and then publishes the new state as nostr event which can be consumed by nostr clients.

The distributed consensus layer makes state immutable by putting a merkle root into the Bitcoin chain.

### Event Kinds
Nostrocket uses event kinds 640000 to 649999.

Nostrocket is composed of multiple state machines called Minds. More can be added at any time.

Event kinds are broken down into `64` indicating a Nostrocket event, followed by two more integers indicating the Mind, and another two indicating sub-state or action type within that Mind.


| Description            | Nostrocket  | Mind | State or Action |
|------------------------| ------------- |---------------|-----------------|
| Lead Time Adjustment   | 64  | 02 [Shares]   | 00              |
| Share Transfer         | 64  | 02            | 02              |
| New Mirv Cap Table     | 64  | 02            | 08              |
| Votepower Signed State | 64  | 00 [Consensus] | 64              |
| Add Identity to Tree   | 64  | 04 [Identity] | 02              |
| Add OP_RETURN Address  | 64  | 04            | 06              |

#### Event Structure
JSON in content is a bad idea, but I didn't think that through before I started doing it.

There is some legacy code here which uses JSON in content.

I'm migrating this to tags.

The Content field will be reserved for human-readable text from the user who creates an event.

#### Replay Protection
`["r", "<last successful state-change event from this pubkey>"]`   
Events which do not have replay protection MUST NOT cause a state change.

#### Bitcoin 
`["btc", "<height int decimal>", "<header 32 bytes hex>", "<unix timestamp int decimal>"]`


### Terminology
**Mirv:** A Nostrocket Mirv is an independent project launched using Nostrocket.