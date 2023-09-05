# Snub
### I didn't choose the snub life; the snub life chose me.
Github did something to me recently which really annoyed me, and its become obvious that storing canonical git repositories on centralised infrastructure is no longer a viable approach for open source projects.

This is especially if you're working on freedom tech. In any case, it's not going to work for Nostrocket, so I have to build something that will solve the problem.

![](https://image.nostr.build/b87d253e485246c32a4e062d21db046994daba3617d8a4c9bf60ceb3de05adc5.jpg)

## Git *over* nostr is not the solution
The git **over** nostr approach that I keep seeing people take just sends patches around. We can already do that with email, and there's a reason we don't. This is not a feasible approach to building git **on** nostr.

Having to iterate over 38,000 individual patches every time you want to rebuild a git repo locally is not fun. This is why git doesn't work that way internally, and this is one of many reasons why sending patches over nostr doesn't really accomplish anything.

We need to put git internals **directly** onto nostr, rather than take shortcuts by relying on your local computer or a git server to rebuild git internal objects from some combination of patches that exist as events.

The "patches over nostr" approach is also going to be basically useless in terms of allowing people to build github type clients to interact with "git" repositories that don't exist as native git objects. 

People are sometimes confused because commits are **represented** on github etc like a patch, but git commits do **not** contain patches, and there's a very good reason for that, which becomes even *more* important when we are talking about repositories existing as events.

## The End Goal for Snub
Censorship resistant canonical git repositories that can be directly consumed by nostr clients.
I want my git repositories to be:
* as censorship resistant and as trivially redundant as my nostr notes
* interaction using public/private keypairs ONLY
* Web clients can read git objects (blobs, trees, refs, etc) directly from events, they don't need to rebuild the entire repo from patches to know what the current state is,
* totally open and permissionless, anyone can easily build microapps that interact directly with git internals from events to create whatever frontend features they want
* NO F*CKING SERVERS!
* NO F*CKING DOMAIN NAMES
* NO F*CKING usernames and passwords, personal access tokens, ssh certificates, etc etc
* lightning as a first class citizen
* Fully compatible with existing git tooling


## Current Status
You can publish a git repository as events. I've spent about a week on getting this part working. Next up I will implement clone. I'm actively working on this whenever I've got a spare minute.

## Get Started
Install the latest version of Golang, and make sure your `go/bin` directory is in your PATH. Basically just follow the usual golang install instructions. Then:

```
git clone https://github.com/nostrocket/engine.git
cd engine
make installsnub
snub init //optional
snub publish
```

After that, to view some example events that you just published:

```
snub example -a "31228:<your pubkey>:dTag //this information will be printed after publishing
```

For configuration options, look in `~/nostrocket/config.yaml` and `<repo>/.snub/config.yaml`

To use a different pubkey other than the one that was automatically generated, modify the pubkey and private key in `~/nostrocket/wallet.dat`

### Using a local relay
This is highly recommended for the moment, don't really want to spam everyone's relays.
```
git clone https://github.com/nostrocket/flamebucket.git
cd flamebucket
./install.sh
./start_local_relay.sh
```

## Architecture
### Repo Anchor - Kind 31228
This is a replaceable event that MUST contain the repository name and D tag.   
It MUST contain either a list of maintainer pubkeys OR a Rocket ID to take maintainers from a Nostrocket organization.   
It MAY contain an `a` tag identifying an upstream repository if this is a fork.   
It MAY contain a `forknode` tag identifying the latest commit which is at parity with the upstream repository.   

### Branch - Kind 31227
This is a replaceable event that MUST contain a branch name, HEAD ref, `d` tag, and `a` tag identifying the kind 31228 Repo Anchor.

### Commit - Kind 3121
This is a non-replableable event containign a git Commit object. 
#### MUST contain:
* a SHA1 git identifier in the `gid` tag,
* a SHA1 git identifier in the `tree` tag,
* the `d` tag of a Repo Anchor event in the `a` tag
#### MAY contain:
* a list of SHA1 git identifiers of parent commits in the `parents` tag,
* an author name, email, unix timestamp, and UTC offset value in the `author` tag (author and committer not necessary except for maintaining the same hashes used by legacy platforms)
* a committer name, email, unix timestamp, and UTC offset value in the `committer` tag
* a commit message in the event Content
* a nonce in the `nonce` tag to mine an event ID which starts with the same characters as the SHA1 git identifier for this commit

### Tree - Kind 3122
This is a non-replaceable event containing a git tree object.
#### MUST contain:
* a SHA1 git identifier in the `gid` tag,
* the `d` tag of a Repo Anchor event in the `a` tag
#### MAY contain:
* a list of git blob identifiers in the `blob` tag, of the format <SHA1>:<FileName>:<FileMode>
* a list of git tree identifiers in the `tree` tag, of the format <SHA1>:<DirectoryName>:<FileMode>
* a nonce in the `nonce` tag to mine an event ID which starts with the same characters as the SHA1 git identifier for this commit

### Blob - Kind 3123
This is a non-replaceable event containing a git blob object.
#### MUST contain:
* a SHA1 git identifier in the `gid` tag,
* the `d` tag of a Repo Anchor event in the `a` tag
* the blob data in the `data` tag. This is currently the binary blob compressed with gzip (for browser speed) and encoded to hex.
#### MAY contain:
* a nonce in the `nonce` tag to mine an event ID which starts with the same characters as the SHA1 git identifier for this commit

### Merges and Pull Requests
A Fast Forward merge is straightforward: you publish a new HEAD on the `master` branch to include the latest commit.
Any other kind of merge requires a new commit to combine multiple commits and possibly resolve conflicts, this is simply a commit object which may or may not update blobs and trees.

A commit that tags a branch `d` tag is a pull request.

Clients SHOULD follow the list of maintainers to find the latest `master` branch. Maintainers SHOULD update their branches whenever another maintainer merges a commit. In any case, the latest HEAD of a branch can be found by subscribing to kind `31227` events from all maintainers and selecting the most recent (the "longest chain").

## Event Examples
``` 
-----THIS IS A REPOSITORY ANCHOR EVENT-----
nostr.Event{ID:"a45e2ecbe3c543421f871cb39ea072defc1ab2274fa3c77bdaed34d6e7621b8e", PubKey:"546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882", CreatedAt:1693915528, Kind:31228, Tags:nostr.Tags{nostr.Tag{"rocket", ""}, nostr.Tag{"name", "engine"}, nostr.Tag{"d", "f770853bd251c4a9c8ffcc34fa19fea11d882b70c5f0e502182e6bbc01233db6"}, nostr.Tag{"a", ""}, nostr.Tag{"forknode", ""}, nostr.Tag{"maintainers", "546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882"}}, Content:"", Sig:"af7762f523855ae853a593f776611ee6da5164e2d1e228e10072fbfd79ea26a1cb56f6b51b4fb2c051b1a955aaf2746dcb1548fff40a9d83b3c9d670730fb36f"}


-----THIS IS A BRANCH EVENT-----
nostr.Event{ID:"7aa9c0b784a12d7a0612b4e768ef4ebcb948c391600a28be7bfaf3ecaf28b2ae", PubKey:"546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882", CreatedAt:1693915523, Kind:31227, Tags:nostr.Tags{nostr.Tag{"name", "main"}, nostr.Tag{"d", "4d67f1338d1951d31d22b5d66ab0aa75f8840679cd1a7bc838bd5cac3d0936c1"}, nostr.Tag{"head", "1fad6be89e30c672d364f70e118c649e15b5dd2b"}, nostr.Tag{"a", "31228:546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882:f770853bd251c4a9c8ffcc34fa19fea11d882b70c5f0e502182e6bbc01233db6"}, nostr.Tag{"len", "159"}}, Content:"", Sig:"06044ff2d3a0868a7734e6bb2a26ad24548eac3ed83f58b625f775926bed70318539957171555c9ae3a12bfbce8192ddb1bb7ceec52ef53a3088ad331ef3490f"}


-----THIS IS A TREE EVENT-----
nostr.Event{ID:"9530fd6c726067095cd5849bfc401fface65d8532adb36e60902f95ac70e8dc2", PubKey:"546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882", CreatedAt:1693915462, Kind:3122, Tags:nostr.Tags{nostr.Tag{"gid", "95307d427af99a5c217ee6ce37f15858392ab051"}, nostr.Tag{"a", "31228:546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882:f770853bd251c4a9c8ffcc34fa19fea11d882b70c5f0e502182e6bbc01233db6"}, nostr.Tag{"blobs", "c081adc92552ce5777a860bd71b398ccacfe5a38:handlers.go:100644", "f586d6471181b439c55e15cf267499ab8a595083:state.go:100644", "f1ea8b2a05bf208c355f07110f411edb91d2151c:types.go:100644"}, nostr.Tag{"trees"}, nostr.Tag{"nonce", "12633"}}, Content:"snub tree object", Sig:"40d147a7b24a2f9f32d859d6d0588258fc888cbe111c9a9e81099e0cd3fdebbe2ba3ec975459d9a7b0b1b51bcc5d5245e805c27f68a53201af6acf3a0ffe5fce"}


-----THIS IS A COMMIT EVENT-----
nostr.Event{ID:"c55a5298a4856dbc8d3348e214ffb93914feac1c699158dd77b0293c67faef64", PubKey:"546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882", CreatedAt:1693915445, Kind:3121, Tags:nostr.Tags{nostr.Tag{"gid", "c55aca6c5755c4bd7c96edec71b666aa53710c0e"}, nostr.Tag{"tree", "ee86287a0d3643e038418b03b41152636c1859d5"}, nostr.Tag{"a", "31228:546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882:f770853bd251c4a9c8ffcc34fa19fea11d882b70c5f0e502182e6bbc01233db6"}, nostr.Tag{"author", "gsovereignty", "gsovereignty@proton.me", "1680490274", "+0800"}, nostr.Tag{"committer", "gsovereignty", "gsovereignty@proton.me", "1680490274", "+0800"}, nostr.Tag{"parents", "b9827ba53b7dfb62be2b23c631a3ca2131f5a38d"}, nostr.Tag{"nonce", "36662"}}, Content:"Problem: errors are swallowed\n", Sig:"e057fd05bb209762e3f1b2f7597e2ec84af52d591b816de410fd9a8110acae1d1603ea18c059fc5f30cb0060b6b37eca5f1020b5b3e7dc6be1b9af416f020852"}


-----THIS IS A BLOB EVENT-----
Binary blobs are compressed and hex encoded to make events smaller, you can see this data in the data tag.

nostr.Event{ID:"201bac1868060ed939b275d342ea2309517a15a9b0c846cfa1e838fa59c23140", PubKey:"546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882", CreatedAt:1693915470, Kind:3123, Tags:nostr.Tags{nostr.Tag{"gid", "201b148a0c2ec5c2a80dd52eed847f2a47c33205"}, nostr.Tag{"data", "1f8b08000000000000ff8c58ff6fdbb612ff59fa2b6e068a67a78954274bda064986b469b300ed5eb1a4fb250b1e68e924f3592205f214cfd8fabf0f4752b2ec385b8520b678e47dfddcf1ce8dc816a244b0aa9dc5b1ac1b6d08c671342a6a1ac571342a25cddb5992e93a2df54129a9fb783c1efd33396daab69e4955a685acb0d6396e1d50b3fc6049059f50da92d922dba6981ea58fb24147715b74b6404a519552615ac9991166358a27714cab06e1576c3458326d46f0671c5daa6cae0d805bf72f71f45ed7b5240b508be63e70486ee762fae029717467102dc08e1d4c89a37746a86c8ed6d12d19a9ca07bf06697a7f36f35f95a8f122acc7d1bb4acf76f3644a1c5d4b02f794921256d74ad266c5eaaa429600b0e71c91fcc6ffe3e816558e6c5a36170a9c6b920f8fa828fe16c745ab32181bd8634613a8b4c83f1a5d5f49bb184f008dd186dd637a31fbbc08a7e7a01b546be9639378af259f7426aa2b692671240bb7f9877350b2623691416a8de2d538fa1647266163ce61cf0ccc087b94ac58c134851b2509328382d08280cc99d91a41522b102a775a5ba039c29a0f1446d7904bbb00a9488313b56d2e731e9a194c33c9a61fbec312a6f241a924f9387cd7b14d5b1d303db406d04cd3b95e42ae61895022411650e90c440ea4755e6884b1c85ea8bdc5a524d0b3ff6346f627980b3b3ff06f3f314b2666820e38dde0a081bc3816c574fa635e4cdfbe3e7a9567d3fcf0edd1091ebf99bec557af0f5f1f4ddf1c8ad727007f39c6037e70d0e90407079672a9e2e8fae60afa6788e138ba6cc9671a3f9fb014d9ea264745b290990f699ab6167328b481cad1196b3edc16967354d0b4552555e95dc0eab8b8efb38772adfe43a01073104ed23e48e54a963f5aa329f9282d356803b536c89e735e5c226442814584e55c3b40791e1684412f8d17c3feae4090cbaee76c7107d058ad1cd3a69d55d2724960820b5f1c7d466bb9b2bac7978938fae2a4dc5cb95a70ffb0e9452e2fbd8f37492eb73bda8074787c12475ec95b57267a497ef19dc8166d131683e2462c81f00f025d387d439c3df4fe9096782b3b1f682e085a8b16b016b2b2202cdc5c39605e7fb9062b4bbbae3719ec79d74de0d6891b4f606c83e80983dec2cb731891418411bc842c0906bf84d1ef6a14470c8effed43eed255a812214bd60ee34cf31c7ca81c8fbc3ffc2df0cf120fc6c4764a743b02b98fefd31db2800ad5384b7a8f4ee0025e0d44974d696519d4ef77ada56f480a08e82a425f0d76826a5d1b7e11f5166a3eb0f73756ee648d9644dd705938f9318ebedebdd7456191d65b58d4e010a4a9073efc15424e68d6e1ab606f975e13e8bd14d8fcd957b8a2a6e4b6315251311ebdb0f0c2c2d90b7b012f72786147fb5025ac037fb249fce90c71844e7f7ee9759f7c873a241cb0285c7a77c2638b6f1cd134a8f231fd83e44d95f3d1862e932d65b6e3c670dda8e1ad45a8170ed052594291734e0dca681cdd10d6d6e5ba033b61bd2ea4bb537c2bbb7b8f10ec318b67b32b648f24acd709448957a047f08e98b9bfdfd5c89f4d3e86662de9e48475ef51f7f56761e7e12bfb7732b8f6369cc5b2070eeb3843d70f3a599f758e03d077f065193bbcd4a33aecdb760f8b9cf40236501bb29b9e5ac8597e32bcca77ecb99f9e3e0caff71d5b426bb33492f052e5bf894ae682d02ff8bbc16185ebe763205a5f6425412d28ebae1046c80c59efac12b2c67c1b03db32860dcfa33050498516ee1f3a6ffe2b380a1798d3731fd5b5ffe228f2bcfaf472afdb99d4a1885ce23b66ff0e1b8f1b3b17d3be01f5d9e465ececb558ec17965a7124fb003eedc2640194b01f7f380716310c2f33f9c0ee2ac6a342c80a73e06e121ba3f33643a7e27ee77a170ce90c9cb5e4fb892a6b2b8eac243e38435fef9cbc7d276df25c271806855dd57e58ab9f7dd294070b7f754b0b61d8900a9a4a48c5d7fa3e7cfe7a7b074a7307a74870a7d4880c6d1cfdcc156a770fb75b56680d64a8c13c795cde89a17a7d157e56dfd05cf5edc6488cb8887bdc375a729346baeff441f8a9ed687a78f8e6f4ac69670b5cf159477413833617a767eed5b55a9033bf8b38bada5463b390ee52cd0895ebda15ecaeef0b55d86e0f6a87c7271bdd1adfa65049ebccf26af029ce335155ae23c57cddd66bc3c17844255165e8bbd602299b73b07dbfdf69702d437ff864547cdaf7a992e6bd3dae13080e274da202d5d63334ac61a7885443dcc4d12761e96be3aad4068bbebdd00530d22d41ebb6f5405ecfd40330bf77035dfe6e35f4ff6596e9963be20da877956923684f43b615a4af8d2583a276a79ee68a37be0d9b6038e16a55adc0cd2032e48ee4c1b3d066b1cfff6b413e7166b889bd8bd333967611471fb559607e49cf248f171e52c649580a17fa05cf2d3c1170860aa9d0d861f31ffc0369baa1e32fffbde3786069d8a5b0943487f56f2071f4abfbdc729c5fe4fe628bdbf39c38f97804ad07da2165fd6464700b261e245b6e7f1e3051f7e3c120ec4f6356e9d0076b05dd7be5ee71bbb27c6b755173138af5f46ab5aeab959e0d8078bd63569afa9f61ae0409b87f98ad58b77eae7ada75f56c3f8be6999f6dfe0e0000ffff46627f8c3d130000"}, nostr.Tag{"a", "31228:546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882:f770853bd251c4a9c8ffcc34fa19fea11d882b70c5f0e502182e6bbc01233db6"}, nostr.Tag{"nonce", "8685"}}, Content:"snub blob object", Sig:"0bc3ce55a2945b6e2fa686280985c2bdf85c9b2fc4562cc4590f421f1cdede54ec876c9fced45f0cc3167180803fe9eb5de31373bf30d6d7c4216031f77d628f"}

```
