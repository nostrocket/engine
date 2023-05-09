# Nostrocket Problem Tracker

## Event Structure
### Anchor Event
The purpose of an anchor event is to lock the hierarchical structure of the problem tracker so that problems don't become orphaned by moving the parent.

To log a new problem, create an Event of Kind 641800 (an *anchor* event) containing the following tags:  
```
//standard nostrocket tags
["e", "<nostrocket ignition event>", "", "root"]
["e", "<ignition problem>", "", "reply"]
["e", "<parent of this problem>", "", "reply"] //only if not the ignition problem
```

The content field MAY contain a plain text summary of the problem to be rendered in any Nostr client.

### Problem Event
A problem event is an object which contains the problem title, body, and metadata.

A problem MAY be modified by the creator of the Anchor event or a Maintainer, this is done by producing a new problem event in reply to the same anchor event, and including tags for whatever the person wants to update.

Clients SHOULD rebuild the current problem state by iterating through each problem event signed by the creator of the anchor event, or a maintainer of Nostrocket or the relevant Mirv.

A problem event MUST contain the following tags to be parsed by Nostrocket:

``` 
["e", "<nostrocket ignition event>", "", "root"]
["e", "<anchor event ID>", "", "reply"]
```

A problem event MUST contain a tag with a sequence number so that old events don't accidentally get used in clients. 

The first problem event MUST have a sequence of `1` and all subsequent problem events MUST be the latest event's sequence + 1.

``` 
["sequence", "<current sequence + 1>"]
```

A problem event MAY contain a title tag of less than 70 characters in plain text. 

``` 
["title", "<plain text content>"]
```

A problem event MAY contain a body tag to describe the problem using markdown

``` 
["body", "<markdown content>"]
```

A problem event MAY contain a tag to indicate if the problem has been claimed or not.

Any pubkey in the Identity Tree MAY create a problem event containing this tag to claim an event to work on.

Clients SHOULD ignore problem events containing a `claimed_by` tag if the is an existing claim.

```
["claimed_by", "<pubkey>"]
```

To remove a claim, a Maintainer MUST produce a problem event containing an empty string:   
```
["claimed_by", ""]
```

The problem creator, or a Maintainer MAY close the problem.

A problem event MAY contain a `status` tag to indicate if the problem is open or closed. If this tag does not exist, the problem is considered open.


```
["status", "closed"] //to close the problem
["status", "open"] //to re-open the problem
```

### Comments and votes
To comment on a problem, a participant SHOULD reply to the anchor event with a Kind 1 event.

This event MUST include the anchor event as the root
```
["e", "<anchor event ID>", "", "root"]
```

If the reply is to another comment in the problem thread, this should also be included   
```
Tags: 
["e", "<anchor event ID>", "", "root"]
["e", "<parent comment ID>", "", "reply"]

Content: the comment.
```

Participants with votepower in the relevant Mirv MAY indicate if they believe this problem is ready to be solved (and should be solved).

```
["e", "<anchor event ID>", "", "root"]

Content: +
```

Clients SHOULD find the pubkey in the Mirv cap table (produced by the Nostrocket Engine and published as an event) and add up the total votepower that has indicated.