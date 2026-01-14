# cmus-lyrics

A TUI application that displays lyrics for the currently playing song in cmus.

A Genius API access token is required. See their [API
documentation](https://docs.genius.com/) for information on how to get one. The
program reads the access token from the file
`~/.config/lyrics/config.json`. The config file should have the following
content:

```
{
  "genius_access_token": "YOUR_TOKEN_HERE"
}
```
