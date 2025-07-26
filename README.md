![GolangCI](https://github.com/marcopaganini/videofix/actions/workflows/golangci-lint.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/marcopaganini/videofix)](https://goreportcard.com/report/github.com/marcopaganini/videofix)

# videofix - Fix common problems in your MKV files.

## Description

`videofix` automatically fixes common problems in your MKV files, such as:

* Transcode EAC3 tracks to AAC, which is universally supported.  Existing EAC3
  tracks will be removed if a corresponding AAC track already exists (same
  language).
* Re-order tracks: tracks are re-ordered so that the output file contains
  video, audio, and subtitle tracks, in this order.
* Sets tracks of your preferred language as default tracks.
* Removes the default flags on all other tracks.
* Optionally removes all tracks that don't match a specified language.

## Installation

### Automatic process

To download and install automatically (under `/usr/local/bin`), just run:

```bash
curl -s \
  'https://raw.githubusercontent.com/marcopaganini/videofix/master/install.sh' |
  sudo sh -s -- marcopaganini/videofix
```

This assumes you have root equivalence using `sudo` and will possibly require you
to enter your password.

To download and install under another directory (for example, `$HOME/.local/bin`), run:

```bash
curl -s \
  'https://raw.githubusercontent.com/marcopaganini/videofix/master/install.sh' |
  sh -s -- marcopaganini/videofix "$HOME/.local/bin"
```

Note that `sudo` is not required on the second command as the installation directory
is under your home. Whatever location you choose, make sure your PATH environment
variable contains that location.

### Homebrew

`videofix` is available on homebrew. To install, use:

```
brew tap marcopaganini/homebrew-tap
brew install videofix
```

### Manual process

Just navigate to the [releases page](https://github.com/marcopaganini/videofix/releases) and download the desired
version. Unpack the tar file into `/usr/local/bin` and run a `chmod 755
/usr/local/bin/videofix`.  If typing `videofix` doesn't work after that, make sure
`/usr/local/bin` is in your PATH. In some distros you may need to create
`/usr/local/bin` first.

### Using go

If you have go installed, just run:

```
go install github.com/marcopaganini/videofix@latest
```

## Using videofix

Usage is simple:

```bash
videofix [options] mkvfile.mkv
```

The program will use a temporary file on the same directory (and refuse to
proceed if that file already exists).  Once the process is done, it will
replace the original file.

Options:

* `--lang`: language of the default audio and subtitle tracks. This will cause
  `videofix` to set the default flag on all tracks that match the default
  language, and remove it on all tracks that don't.  This makes it easier for
  players to start automatically on your preferred language.

* `--prune`: When combined with `--lang`, this will cause the removal of all
  tracks that are not in your preferred language. The program will refuse to
  proceed if this will result in the complete removal of a given track type
  (like audio or subtitle). Use with care.

## Contributions

Feel free to open issues, send ideas and PRs.
