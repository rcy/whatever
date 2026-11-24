# whatever

An app that does things

## Install

```sh
go install github.com/rcy/whatever@latest
```

## Commands

### Notes

Add, show, delete and undelete notes.

```sh
# create a note
$ whatever notes add think about going outside

# notes can be anything
$ whatever notes add https://www.youtube.com/watch?v=I8jfn8k8vpM

# show all notes
$ whatever notes
9820beb 2025-09-06 12:27:54 think about going outside
403361a 2025-09-06 12:28:31 https://www.youtube.com/watch?v=I8jfn8k8vpM

# delete a note (unambiguous prefixes of the id work)
$ whatever notes delete 982

$ whatever notes
403361a 2025-09-06 12:28:31 https://www.youtube.com/watch?v=I8jfn8k8vpM

# you can undelete a note
$ whatever notes undelete 982

$ whatever notes
9820beb 2025-09-06 12:27:54 think about going outside
403361a 2025-09-06 12:28:31 https://www.youtube.com/watch?v=I8jfn8k8vpM

$ whatever edit 403 cooontent
$ whatever notes
9820beb 2025-09-06 12:27:54 think about going outside
403361a 2025-09-06 12:28:31 cooontent

# show all events in the log
$ whatever events
1 9820b NoteCreated     2025-09-06T19:27:54Z {"Text":"think about going outside"}
2 40336 NoteCreated     2025-09-06T19:28:31Z {"Text":"https://www.youtube.com/watch?v=I8jfn8k8vpM"}
3 9820b NoteDeleted     2025-09-06T19:29:07Z null
4 9820b NoteUndeleted   2025-09-06T19:29:29Z null
4 9820b NoteTextUpdated 2025-09-06T19:29:29Z {"Text":"cooontent"}
```

### Discordian Date

```sh
$ whatever ddate
Prickle-Prickle, Bureaucracy 30, 3191 YOLD
```

### Web Server!

```sh
$ whatever serve
listening on http://localhost:9999
```

#### MCP server

A read-only [MCP](https://modelcontextprotocol.io) endpoint is served at
`/mcp` with `OWNER_ID`'s notes. Tools: `notes_overview`, `list_notes`
(search/filter), `get_note`.

It is protected by a built-in OAuth 2.1 authorization server (dynamic client
registration, PKCE) that delegates login to Google; only `OWNER_ID` can
authorize. Tokens are signed with `SESSION_SECRET`, so rotating it revokes
them all. `BASE_URL` must be the public URL.

```sh
# Claude Code
claude mcp add --transport http whatever https://<host>/mcp
# then run /mcp in Claude Code to log in

# claude.ai: Settings → Connectors → Add custom connector → https://<host>/mcp
```

### Reporting bugs, feature requests, or whatever

```sh
$ whatever bug
```
