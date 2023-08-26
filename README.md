# Nostrocket Engine
Problem: can't reach consensus about the current state of Nostrocket and Subrockets

This repo doesn't have an issue tracker because we are using Nostrocket to track problems. Please view the problem tracker using [Spacemen](https://nostrocket.github.io/spaceman/)

### Nostrocket Engine is a replicated state machine built with Nostr and Bitcoin.

It's essentially a "Layer 2" which parses nostr events, and updates state if the event complies with the protocol.

The distributed consensus layer makes state immutable by putting a merkle root into the Bitcoin chain.

#### Replay Protection
`["r", "<last successful state-change event from this pubkey>"]`   
Events which do not have replay protection MUST NOT cause a state change.

#### Bitcoin Block Example
Nostrocket requires visibility into the Timechain and Truthchain. For timing, we publish bitcoin blocks as events, and Nostrocket clients can validate these against their own preferred block source (Core, blockstream API, etc).
```
nostr.Event{
    ID:"8a31497568b889fefd0f1acc7d8f3a2537d330ad9d73814056773488ef6b5216", 
    PubKey:"60ad82c605f8c78f86afea199a56ba713fbd214584d63fa6b11466eb9ceaf7fb", 
    CreatedAt:1693054709, 
    Kind:1517, 
    Tags:nostr.Tags{
        nostr.Tag{"e", "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8"}, //nostrocket ignition event
        nostr.Tag{"hash", "000000000000000000036edbb338767710e6ce191f36ffbd197056cd3c667d57"}, 
        nostr.Tag{"height", "804934"}, 
        nostr.Tag{"difficulty", "55621444139429"}, 
        nostr.Tag{"minertime", "1693053194"}, 
        nostr.Tag{"mediantime", "1693052179"}}, 
    Content:"", 
    Sig:"bd7261a62efbb6eb2835a8643d3a03b38a2c6ae658f28dd541ec34ac8fd9c4b39804c2cdd11812fef848a0f1b9dd24dbe8b86df1ac26a8e64e8128cd34859eaa"
```

### Running Nostrocket Engine
Ensure you have the latest version of Golang installed for your system, and run:
`make`

To publish blocks:
`make blocks`