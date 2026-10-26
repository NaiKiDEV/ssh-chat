# ssh-chat

A chat server you connect to with plain `ssh`.

## Run

```sh
go run .
```

Then connect from any terminal:

```sh
ssh -p 23234 yourname@localhost
```

A host key is generated at `.ssh/id_ed25519` on first start. Requires Go 1.26.

## Identity

- Connecting with an SSH key ties your profile (name, color, time zone) to that key. It is restored next time and nobody else can take your saved name.
- Connecting without a key makes you a guest (`~name`). Guest settings last until you disconnect. Disable guests with `-allow-guests=false`.

## Using it

Lobby: pick a listed room with up/down and press enter, or type any room name to join or create it. Typed rooms are unlisted and disappear when the last person leaves.

| Key | Action |
| --- | --- |
| enter | send |
| ctrl+enter (also shift+enter, alt+enter where the terminal allows) | new line |
| pgup / pgdn, mouse wheel | scroll |
| esc | scroll mode (arrow keys scroll, enter or typing returns to the input) |
| ctrl+s | profile settings |
| ctrl+c | quit |

Names use `a-z`, `A-Z`, `0-9`, `-`, `_` and `.` so they cannot be faked with look-alike characters. Pink is reserved for your own name.

| Command | Action |
| --- | --- |
| `/me <action>` | emote |
| `/nick <name>` | change name |
| `/join <room>` | switch room |
| `/leave` | back to lobby |
| `/settings` | profile settings |
| `/quit` | disconnect |
| `//text` | send a message starting with `/` |

## Configuration

Every flag can also be set with an `SSH_CHAT_` environment variable, e.g. `SSH_CHAT_PORT=2222`.

| Flag | Default | |
| --- | --- | --- |
| `-host` | `localhost` | listen address, use `0.0.0.0` to expose |
| `-port` | `23234` | |
| `-host-key` | `.ssh/id_ed25519` | |
| `-profiles` | `.data/profiles.json` | empty keeps profiles in memory |
| `-rooms` | `public` | comma separated listed rooms |
| `-history` | `200` | messages kept per room |
| `-max-rooms` | `100` | |
| `-max-users` | `200` | concurrent connections |
| `-allow-guests` | `true` | |
| `-idle-timeout` | `1h` | `0` disables |
| `-log-level` | `info` | |

## Limits

Messages are limited to 1000 characters and 20 lines, and each connection can post about 2 messages per second (burst of 6). New connections are rate limited per IP. All user text is stripped of terminal control sequences before anyone sees it.
