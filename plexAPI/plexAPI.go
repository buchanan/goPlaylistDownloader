package plexAPI

import (
	"log"
	"net/http"
	"io/ioutil"
	"encoding/xml"
	"fmt"
	"strings"
	"encoding/base64"
	"os"
	"io"
	"strconv"
	"time"
)

type PipeViewer struct {
	io.Reader
	AmountRead float64
	Total float64
	count float64
	timer time.Time
}

func (pv *PipeViewer) Read(p []byte) (int, error) {
	n, err := pv.Reader.Read(p)
	pv.AmountRead += float64(n)
	pv.count += float64(n)

	if since := time.Since(pv.timer); since > time.Second {
		var complete float64 = (pv.AmountRead/pv.Total)*100
		var seconds float64 = float64(since)/1000000000
		var Mbps float64 = (pv.count/seconds)/1048576

		pv.count = 0
		pv.timer = time.Now()

		if err == nil {
			fmt.Printf("\r\033[K%.2f percent complete %.2f Mbps", complete, Mbps)
		}
	}

	return n, err
}

type Account struct {
	Email string
	Token string
	username string
	password string
	Devices []Device
	Authenticated bool
	failCount int
	failTimestamp time.Time
}

func (a *Account) Fail() {
	switch a.failCount {
	case 2: if time.Since(a.failTimestamp) <= 10*time.Minute { log.Fatal("Failed 3x within 20 min") } else { a.failCount--;a.failTimestamp=time.Now() }
	case 1: if time.Since(a.failTimestamp) <= 10*time.Minute { a.failCount++ };a.failTimestamp=time.Now()
	case 0: a.failCount++;a.failTimestamp=time.Now()
	}
}

func (a *Account) Login(user,pass string) bool {
	a.username = user
	a.password = pass
	a.Authenticated = a.authenticate()
	return a.Authenticated
}

func (a *Account) authenticate() bool {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://plex.tv/users/sign_in.xml", nil)
	req.Header.Add("Authorization", "Basic "+base64.URLEncoding.EncodeToString([]byte(a.username+":"+a.password)))
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
		log.Println("Unable to connect to plex.tv", err)
		a.Fail()
		return false
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Unable to read from body", err)
		a.Fail()
		return false
	}
	resp.Body.Close()
	var authContainer struct{
		XMLName xml.Name `xml:"user"`
		Email	string `xml:"email,attr"`
		Token	string `xml:"authenticationToken,attr"`
		Email2	string `xml:"email"`
		Token2	string `xml:"authentication-token"`
	}
	err = xml.Unmarshal(body, &authContainer)
	if err != nil {
		log.Println("Unable to parse message body ", err)
		a.Fail()
		return false
	}
	if authContainer.Email != authContainer.Email2 || authContainer.Token != authContainer.Token2 {
		log.Println("Error email or tokens do not match")
		a.Fail()
		return false
	} else {
		a.Email = authContainer.Email
		a.Token = authContainer.Token
	}

	resp,err = http.Get("https://plex.tv/pms/resources.xml?includeHttps=1&X-Plex-Token="+a.Token)
	if err != nil {
		log.Println("Unable to get devices list", err)
		a.Fail()
		return false
	}
	body, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Println("Unable to read from body", err)
		a.Fail()
		return false
	}
	var DeviceContainer struct {
		XMLName xml.Name `xml:"MediaContainer"`
		Devices []Device `xml:"Device"`
	}
	if err = xml.Unmarshal(body, &DeviceContainer); err != nil {
		log.Println("Unable to parse device list xml", err)
		a.Fail()
		return false
	}
	for di := range DeviceContainer.Devices {
		DeviceContainer.Devices[di].account = a
		DeviceContainer.Devices[di].getPlaylists(false)
		for ci := range DeviceContainer.Devices[di].Connections {
			DeviceContainer.Devices[di].Connections[ci].device = &DeviceContainer.Devices[di]
		}
	}
	a.Devices = DeviceContainer.Devices
	return true
}

type Connection struct {
	device *Device
	Proto string `xml:"protocol,attr"`
	Addr string `xml:"address,attr"`
	Port string `xml:"port,attr"`
	URI string `xml:"uri,attr"`
	Local string `xml:"local,attr"`
}

type Playlist struct {
	device *Device
	connection *Connection
	Key string `xml:"key,attr"`
	Title string `xml:"title,attr"`
}

func (p *Playlist) Download(downloadPath string, depth int){
	defer p.device.account.authenticate()
	// TODO incorperate file depth
	// get list of Videos to download
	resp, err := http.Get(p.connection.URI+p.Key+"?X-Plex-Token="+p.device.Token)
	if err != nil {
		log.Println("Error fetching videos list", err)
		p.device.account.Fail()
		return
	}
	body,err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Println("Error reading from videos body", err)
		p.device.account.Fail()
		return
	}
	var VideosContainer struct {
		XMLName xml.Name `xml:"MediaContainer"`
		Size string `xml:"size,attr"`
		TotalSize string `xml:"totalSize, attr"`
		Title string `xml:"title,attr"`
		Key string `xml:"ratingKey,attr"`
		Videos []Video `xml:"Video"`
	}
	if err = xml.Unmarshal(body, &VideosContainer); err != nil {
		log.Println("Error parsing xml body of videos", err)
		p.device.account.Fail()
		return
	}
	// Loop through movies and download
	for c, v := range VideosContainer.Videos {
		var part Part = v.Media[0].Part
		//TODO select "BEST" video available
		//TODO check if video is "available" Pride and prejudice and zombies
		//for mi, m := range v.Media {
		//	part = m.Part
		//}
		partname := part.File
		fileSize,_ := strconv.Atoi(part.Size)
		var filePath string
		if i := strings.Index(partname, ":\\"); i > -1 {
			filePath = partname[i+2:]
		}
		filePathS := strings.Split(strings.Replace(filePath, "\\", "/", -1), "/")
		filename := filePathS[len(filePathS)-1]
		parentFolder := filePathS[len(filePathS)-2]
		if v.Class == "episode" {
			parentFolder = filePathS[len(filePathS)-3]+"/"+parentFolder
		}
		log.Printf("%d/%s Starting download %s\n", c, VideosContainer.TotalSize, parentFolder+"/"+filename)
		err := os.MkdirAll(parentFolder, 0775)
		if err != nil {
			if ! os.IsExist(err) {
				log.Println("Error creating parentFolder", err)
				p.device.account.Fail()
				return
			}
		}
		fullfilepath := strings.Join([]string{downloadPath,parentFolder,filename}, "/")
		if stats, err := os.Stat(fullfilepath); err == nil {
			if stats.Size() > int64(fileSize) {
				log.Println("File found and larger than video in playlist skipping")
				continue
			} else if stats.Size() == int64(fileSize) {
				log.Println("File found and size matches removing from playlist")
				client := &http.Client{}
				req, err := http.NewRequest(http.MethodDelete, p.connection.URI + p.Key + "/" + v.PlaylistItemID + "?X-Plex-Token=" + p.device.Token, nil)
				resp, err = client.Do(req)
				if err != nil {
					log.Println("Error unable to remove movie from playlist", err)
					continue
				}
				continue
			}
			log.Println("File found but is too small attempting to resume")
			fh, err := os.OpenFile(fullfilepath, os.O_APPEND | os.O_WRONLY, 0600)
			if err != nil {
				log.Println("Unable to open file for writing", err)
				p.device.account.Fail()
				return
			}
			req, _ := http.NewRequest(http.MethodHead, p.connection.URI + part.Key + "?X-Plex-Token=" + p.device.Token, nil)
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
			pv := &PipeViewer{Reader: resp.Body, AmountRead: float64(stats.Size()), Total:float64(fileSize)}
			n, err := io.Copy(fh, pv)
			resp.Body.Close()
			fh.Close()
			if err != nil {
				os.Remove(fullfilepath)
				log.Println("Error writing to file", err)
				p.device.account.Fail()
				return
			}
			if n != int64(fileSize) {
				log.Println("Error file size mismatch after download deleting")
				os.Remove(fullfilepath)
				p.device.account.Fail()
				return
			} else {
				fmt.Println("\nDownload Complete!")
			}
			client := &http.Client{}
			req, err = http.NewRequest(http.MethodDelete, p.connection.URI + p.Key + "/" + v.PlaylistItemID + "?X-Plex-Token=" + p.device.Token, nil)
			resp, err = client.Do(req)
			if err != nil {
				log.Println("Error unable to remove movie from playlist", err)
				continue
			} else {
				log.Println(resp.Status + " Removed movie from playlist")
			}
			continue
		}
		fh, err := os.Create(fullfilepath)
		if err != nil {
			log.Println("Unable to create/open file for writing", err)
			p.device.account.Fail()
			return
		}
		resp, err := http.Get(p.connection.URI+part.Key+"?X-Plex-Token="+p.device.Token)
		if err != nil {
			os.Remove(fullfilepath)
			log.Println("Unable to connect to server", err)
			p.device.account.Fail()
			return
		}
		pv := &PipeViewer{Reader: resp.Body, Total:float64(fileSize)}
		n, err := io.Copy(fh, pv)
		resp.Body.Close()
		fh.Close()
		if err != nil {
			os.Remove(fullfilepath)
			log.Println("Error writing to file", err)
			p.device.account.Fail()
			return
		}
		if n != int64(fileSize) {
			log.Println("Error file size mismatch after download deleting")
			os.Remove(fullfilepath)
			p.device.account.Fail()
			return
		} else {
			fmt.Println("\r100 percent Download Complete!")
		}
		req, _ := http.NewRequest(http.MethodDelete, p.connection.URI+p.Key+"/"+v.PlaylistItemID+"?X-Plex-Token="+p.device.Token, nil)
		resp, err = new(http.Client).Do(req)
		if err != nil {
			log.Println("Error unable to remove movie from playlist", err)
			continue
		} else {
			log.Println(resp.Status+" Removed movie from playlist")
		}
	}
	return
}

type Device struct {
	account *Account
	ID string `xml:"clientIdentifier"`
	LastSeen string `xml:"lastSeenAt,attr"`
	Name string `xml:"name,attr"`
	Device string `xml:"device,attr"`
	Token string `xml:"accessToken,attr"`
	Provides string `xml:"provides,attr"`
	Presence string `xml:"presence,attr"`
	Connections []Connection `xml:"Connection"`
	Playlists []Playlist
}

func (d *Device) getPlaylists(allowRemote bool) bool {
	for _,conn := range d.Connections {
		if conn.Local == "0" || allowRemote {
			resp,err := http.Get(conn.URI+"/playlists/all?X-Plex-Token="+d.Token)
			if err != nil {
				log.Println("Error fetching playlists", err)
				d.account.Fail()
				continue
			}
			body,err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				log.Println("Error reading from body", err)
				d.account.Fail()
				continue
			}
			var PlaylistContainer struct {
				XMLName xml.Name `xml:"MediaContainer"`
				Size string `xml:"size,attr"`
				Playlists []Playlist `xml:"Playlist"`
			}
			if err = xml.Unmarshal(body, &PlaylistContainer); err != nil {
				log.Println("Error parsing body", err)
				d.account.Fail()
				continue
			}
			if len(PlaylistContainer.Playlists) == 0 {
				log.Println("No playlists found on", d.Name)
				return false
			}
			for i, _ := range PlaylistContainer.Playlists {
				PlaylistContainer.Playlists[i].device = d
				PlaylistContainer.Playlists[i].connection = &conn
			}
			d.Playlists = PlaylistContainer.Playlists
			return true
		}
	}
	return false
}