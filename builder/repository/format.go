package repository

type Manifest struct {
	RemotePackages []RemotePackage `xml:"remotePackage"`
}

type RemotePackage struct {
	Path        string      `xml:"path,attr"`
	DisplayName string      `xml:"display-name"`
	Revision    Revision    `xml:"revision"`
	TypeDetails TypeDetails `xml:"type-details"`
	ChannelRef  ChannelRef  `xml:"channelRef"`
	Archives    Archives    `xml:"archives"`
}

type Revision struct {
	Major int `xml:"major"`
}

type TypeDetails struct {
	ApiLevel string     `xml:"api-level"`
	Tag      TypeDetail `xml:"tag"`
	Vendor   TypeDetail `xml:"vendor"`
	Abi      string     `xml:"abi"`
}

type TypeDetail struct {
	Id      string `xml:"id"`
	Display string `xml:"display"`
}

type ChannelRef struct {
	Ref string `xml:"ref,attr"`
}

type Archives struct {
	Archive []Archive `xml:"archive"`
}

type Archive struct {
	HostOs   string          `xml:"host-os"`
	HostArch string          `xml:"host-arch"`
	Complete ArchiveComplete `xml:"complete"`
}

type ArchiveComplete struct {
	Size     int    `xml:"size"`
	Checksum string `xml:"checksum"`
	Url      string `xml:"url"`
}
