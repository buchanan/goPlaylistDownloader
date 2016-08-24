package plexAPI

type Account struct {
	Email string
	Token string
}

type Connection struct {
	Proto string `xml:"protocol,attr"`
	Addr string `xml:"address,attr"`
	Port string `xml:"port,attr"`
	URI string `xml:"uri,attr"`
	Local string `xml:"local,attr"`
}
type Device struct {
	ID string `xml:"clientIdentifier"`
	LastSeen string `xml:"lastSeenAt,attr"`
	Name string `xml:"name,attr"`
	Device string `xml:"device,attr"`
	Token string `xml:"accessToken,attr"`
	Provides string `xml:"provides,attr"`
	Presence string `xml:"presence,attr"`
	Connections []Connection `xml:"Connection"`
}