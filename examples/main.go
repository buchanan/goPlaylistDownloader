package main

import (
	"log"
	"fmt"
	"os"
	"../"
	"regexp"
)

func promptInput(query string) string {
	var response string
	fmt.Printf("%s: ", query)
	fmt.Scanln(&response)
	return response
}

func findNextPlaylist(a plexAPI.Account, patterns []*regexp.Regexp) *plexAPI.Playlist {
	for _, device := range a.Devices {
		for _, playlist := range device.Playlists {
			for _, pattern := range patterns {
				if pattern.MatchString(playlist.Title) {
					return &playlist
				}
			}
		}
	}
	return &plexAPI.Playlist{}
}

func main() {
	log.SetOutput(os.Stderr)
	var playlistPatterns []*regexp.Regexp
	var Account plexAPI.Account
	// Read arguments
	for i := 1; i < len(os.Args); i++ {
		if match := regexp.MustCompile("(^.*):(.*$)").FindStringSubmatch(os.Args[i]); match != nil {
			Account.Login(match[1],match[2])
		} else {
			if rex, err := regexp.Compile(os.Args[i]); err == nil {
				playlistPatterns = append(playlistPatterns, rex)
			}
		}
	}
	if ! Account.Authenticated {
		Account.Login(promptInput("Username"), promptInput("Password"))
	}
	for {
		findNextPlaylist(Account, playlistPatterns).Download("./", 1)
	}
}