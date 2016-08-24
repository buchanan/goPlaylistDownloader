package main

import (
	"net/http"
	"log"
	"io/ioutil"
	"encoding/xml"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"./plexAPI"
	"./plexDownloader"
	"./plexMedia"
	"strconv"
	"path"
	"io"
	"regexp"
)
// run with -c to continue downloads
// run with user:pass:server to skip prompts
func downloadPlaylist(conn plexAPI.Connection, device plexAPI.Device, playlistKey string) error {
	// TODO relogin if download fails
	// get list of Videos to download
	resp, err := http.Get(conn.URI+playlistKey+"?X-Plex-Token="+device.Token)
	if err != nil {
		log.Println("Error fetching videos list", err)
		return err
	}
	body,err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Println("Error reading from videos body", err)
		return err
	}
	var VideosContainer struct {
		XMLName xml.Name `xml:"MediaContainer"`
		Size string `xml:"size,attr"`
		TotalSize string `xml:"totalSize, attr"`
		Title string `xml:"title,attr"`
		Key string `xml:"ratingKey,attr"`
		Videos []plexMedia.Video `xml:"Video"`
	}
	if err = xml.Unmarshal(body, &VideosContainer); err != nil {
		log.Println("Error parsing xml body of videos", err)
		return err
	}
	// Loop through movies and download
	for c, v := range VideosContainer.Videos {
		part := v.Media.Part
		fileSize,_ := strconv.Atoi(part.Size)
		filePath := strings.Split(strings.Replace(part.File, "\\", "/", -1), "/")
		filename := filePath[len(filePath)-1]
		parentFolder := filePath[len(filePath)-2]
		if v.Class == "episode" {
			parentFolder = filePath[len(filePath)-3]+"/"+parentFolder
		}
		log.Printf("%d/%s Starting download %s\n", c, VideosContainer.TotalSize, parentFolder+"/"+filename)
		err := os.MkdirAll(parentFolder, 0775)
		if err != nil {
			if ! os.IsExist(err) {
				log.Println("Error creating parentFolder", err)
				return nil
			}
		}
		if stats, err := os.Stat(path.Join(parentFolder,filename)); err == nil && stats.Size() < int64(fileSize) {
			log.Printf("We already have this movie but file is too small.. ")
			if resumeDownloads {
				log.Println("Continuing")
				fh, err := os.OpenFile(path.Join(parentFolder, filename), os.O_APPEND | os.O_WRONLY, 0600)
				if err != nil {
					log.Println("Unable to open file for writing", err)
					return nil
				}
				req, _ := http.NewRequest(http.MethodHead, conn.URI + part.Key + "?X-Plex-Token=" + device.Token, nil)
				req.Header.Add("Range", fmt.Sprintf("bytes=%d-", stats.Size()))
				resp, _ := new(http.Client).Do(req)
				resp.Body.Close()
				if resp.StatusCode != 206 {
					log.Println("Server does not support resume", err)
					continue
				}
				respRange := resp.Header.Get("Content-Range")
				respRange = respRange[strings.Index(respRange, " "):strings.Index(respRange, "-")]
				rangeStart, _ := strconv.ParseInt(respRange, 10, 64)
				_, err = fh.Seek(rangeStart, 0)
				if err != nil {
					log.Println("Unable to seek file", err)
					continue
				}
				req.Method = http.MethodGet
				resp, _ = new(http.Client).Do(req)
				pv := &plexDownloader.PipeViewer{Reader: resp.Body, AmountRead: float64(stats.Size()), Total:float64(fileSize)}
				n, err := io.Copy(fh, pv)
				resp.Body.Close()
				fh.Close()
				if err != nil {
					os.Remove(path.Join(parentFolder, filename))
					log.Println("Error writing to file", err)
					return nil
				}
				if n != int64(fileSize) {
					log.Println("Error file size mismatch after download deleting")
					os.Remove(path.Join(parentFolder, filename))
					return nil
				} else {
					fmt.Printf("\r100 percent Download Complete!\n")
				}
				log.Println("Removing from playlist")
				client := &http.Client{}
				req, err = http.NewRequest(http.MethodDelete, conn.URI + playlistKey + "/" + v.PlaylistItemID + "?X-Plex-Token=" + device.Token, nil)
				resp, err = client.Do(req)
				if err != nil {
					log.Println("Error unable to remove movie from playlist", err)
					continue
				} else {
					log.Println(resp.Status + " Removed movie from playlist")
				}
				continue
			} else {
				log.Println("Starting over")
			}
		}
		fh, err := os.Create(path.Join(parentFolder,filename))
		if err != nil {
			log.Println("Unable to create/open file for writing", err)
			return err
		}
		resp, err := http.Get(conn.URI+part.Key+"?X-Plex-Token="+device.Token)
		if err != nil {
			os.Remove(path.Join(parentFolder,filename))
			log.Println("Unable to connect to server", err)
			return err
		}
		pv := &plexDownloader.PipeViewer{Reader: resp.Body, Total:float64(fileSize)}
		n, err := io.Copy(fh, pv)
		resp.Body.Close()
		fh.Close()
		if err != nil {
			os.Remove(path.Join(parentFolder,filename))
			log.Println("Error writing to file", err)
			return err
		}
		if n != int64(fileSize) {
			log.Println("Error file size mismatch after download deleting")
			os.Remove(path.Join(parentFolder, filename))
			return nil
		} else {
			fmt.Println("\r100 percent Download Complete!\n")
		}
		req, _ := http.NewRequest(http.MethodDelete, conn.URI+playlistKey+"/"+v.PlaylistItemID+"?X-Plex-Token="+device.Token, nil)
		resp, err = new(http.Client).Do(req)
		if err != nil {
			log.Println("Error unable to remove movie from playlist", err)
			continue
		} else {
			log.Println(resp.Status+" Removed movie from playlist")
		}
	}
	return nil
}

func connectToServer(chosenDevice plexAPI.Device){
	for _,conn := range chosenDevice.Connections {
		if conn.Local == "0" {
			log.Println("Found remote connection. Fetching playlists")
			resp,err := http.Get(conn.URI+"/playlists/all?X-Plex-Token="+chosenDevice.Token)
			if err != nil {
				log.Println("Error fetching playlists", err)
				continue
			}
			body,err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				log.Println("Error reading from playlists body", err)
				continue
			}
			type Playlist struct {
				Key string `xml:"key,attr"`
				Title string `xml:"title,attr"`
			}
			var PlaylistContainer struct {
				XMLName xml.Name `xml:"MediaContainer"`
				Size string `xml:"size,attr"`
				Playlists []Playlist `xml:"Playlist"`
			}
			if err = xml.Unmarshal(body, &PlaylistContainer); err != nil {
				log.Println("Error parsing playlists xml body", err)
				continue
			}
			if len(PlaylistContainer.Playlists) == 0 {
				log.Println("Error no playlists found on server")
				continue
			}
			//TODO find playlists matching patterns
			for i,pl := range PlaylistContainer.Playlists {
				fmt.Printf("%d) %s\n", i, pl.Title)
			}
			var s int
			fmt.Printf("Select playlist to download:")
			fmt.Scanf("%d\n", &s)
			chosenPlaylist := PlaylistContainer.Playlists[s]
			//TODO start downloads from every server available
			if err = downloadPlaylist(conn, chosenDevice, chosenPlaylist.Key); err != nil {
				break
			}
		}
	}
}

func authenticateToWeb(user, pass, server string) {
	fmt.Println(user,pass,server)
	var PlexAccount plexAPI.Account
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://plex.tv/users/sign_in.xml", nil)
	req.Header.Add("Authorization", "Basic "+base64.URLEncoding.EncodeToString([]byte(user+":"+pass)))
	req.Header.Add("X-Plex-Client-Identifier", "Yl7QS2JcD2pmY7IRsGPHvyrAYW2OoA")
	req.Header.Add("X-Plex-Product", "Plex Library EXplorer")
	req.Header.Add("X-Plex-Version", "0.0.001")
	req.Header.Add("X-Plex-Device", "PLEX Indexer")
	req.Header.Add("X-Plex-Device-Name", "PLEX Indexer Master")
	req.Header.Add("X-Plex-Platform", "Linux")
	req.Header.Add("X-Plex-Client-Platform", "Linux")
	req.Header.Add("X-Plex-Platform-Version", "7.0")
	req.Header.Add("X-Plex-Provides", "controller")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Unable to connect to plex.tv", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Unable to read from body", err)
	}
	resp.Body.Close()
	var a struct{
		XMLName xml.Name `xml:"user"`
		Email	string `xml:"email,attr"`
		Token	string `xml:"authenticationToken,attr"`
		Email2	string `xml:"email"`
		Token2	string `xml:"authentication-token"`
	}
	err = xml.Unmarshal(body, &a)
	if err != nil {
		log.Fatal("Unable to parse message body ", err)
	}
	if a.Email != a.Email2 || a.Token != a.Token2 {
		log.Fatal("Error email or tokens do not match")
	} else {
		PlexAccount.Email = a.Email
		PlexAccount.Token = a.Token
	}

	resp,err = http.Get("https://plex.tv/pms/resources.xml?includeHttps=1&X-Plex-Token="+PlexAccount.Token)
	if err != nil {
		log.Fatal("Unable to get devices list", err)
	}
	body, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal("Unable to read device list xml", err)
	}
	var DeviceContainer struct {
		XMLName xml.Name `xml:"MediaContainer"`
		Devices []plexAPI.Device `xml:"Device"`
	}
	if err = xml.Unmarshal(body, &DeviceContainer); err != nil {
		log.Fatal("Unable to parse device list xml", err)
	}
	if server == "" {
		//print out server list
		for i,d := range DeviceContainer.Devices {
			if strings.Contains(d.Provides, "server") {
				fmt.Printf("%d) %s\n", i, d.Name)
			}
		}
		//read selection
		var s int
		fmt.Printf("Select server to connect to:")
		fmt.Scanf("%d\n", &s)
		connectToServer(DeviceContainer.Devices[s])
	} else {
		//find server and connect
		for _,d := range DeviceContainer.Devices {
			if d.Name == server {
				connectToServer(d)
			}
		}
	}
}

var resumeDownloads bool = false
var playlistPatterns map[*regexp.Regexp]struct{} = make(map[*regexp.Regexp]struct{}, len(os.Args)-1)

func main() {
	//TODO auto grab all playlists starting with NB
	log.SetOutput(os.Stderr)
	// Sign in and populate Devices
	var auth []string
	for i := 1; i < len(os.Args); i++ {
		if match := regexp.MustCompile("(^.*):(.*)@(.*$)").FindStringSubmatch(os.Args[i]); match != nil {
			auth = match
		} else if os.Args[i] == "-c" {
			resumeDownloads = true
		} else {
			if rex, err := regexp.Compile(os.Args[i]); err == nil {
				playlistPatterns[rex] = struct{}{}
			}
		}
	}
	if len(auth) > 0 {
		authenticateToWeb(auth[1], auth[2], auth[3])
	} else {
		var user, password string
		fmt.Printf("Username: ")
		fmt.Scanln(&user)
		fmt.Printf("Password: ")
		fmt.Scanln(&password)
		log.Println("Signing in.")
		authenticateToWeb(user, password, "")
	}
	return
}