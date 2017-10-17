# itunes

[![GoDoc](https://godoc.org/github.com/deepilla/itunes?status.svg)](https://godoc.org/github.com/deepilla/itunes)
[![Build Status](https://travis-ci.org/deepilla/itunes.svg?branch=master)](https://travis-ci.org/deepilla/itunes)
[![Go Report Card](https://goreportcard.com/badge/github.com/deepilla/itunes)](https://goreportcard.com/report/github.com/deepilla/itunes)

itunes is a tiny library for extracting RSS feeds from iTunes pages, written in Go.

Have you ever needed to get the underlying RSS feed from a podcast's iTunes page? Have you ever needed to do it in Go? Well this is the package for you.

## Installation

    go get github.com/deepilla/itunes

## Usage

Import the itunes package.

    import "github.com/deepilla/itunes"

Call the ToRSS function.

```go
url, err := itunes.ToRSS("https://itunes.apple.com/us/podcast/s-town/id1212558767?mt=2")
if err != nil {
    log.Fatal(err)
}

fmt.Println("RSS feed is", url) // outputs "RSS feed is http://feeds.stownpodcast.org/stownpodcast"
```

## Licensing

itunes is provided under an [MIT License](http://choosealicense.com/licenses/mit/). See the [LICENSE](LICENSE) file for details.
