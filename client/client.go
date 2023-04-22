package client

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sagan/ptool/config"
	"github.com/sagan/ptool/utils"
)

type Torrent struct {
	InfoHash           string
	Name               string
	TrackerDomain      string
	State              string // simplifiec state: seeding|downloading|completed|paused
	Atime              int64  // timestamp torrent added
	Ctime              int64  // timestamp torrent completed. <=0 if not completed.
	Category           string
	Tags               []string
	Downloaded         int64
	DownloadSpeed      int64
	DownloadSpeedLimit int64 // -1 means no limit
	Uploaded           int64
	UploadSpeed        int64
	UploadedSpeedLimit int64 // -1 means no limit
	Size               int64
	SizeCompleted      int64
	Seeders            int64
	Leechers           int64
	Meta               map[string](int64)
}

type Status struct {
	FreeSpaceOnDisk    int64 // -1 means unknown / unlimited
	DownloadSpeed      int64
	UploadSpeed        int64
	DownloadSpeedLimit int64 // <= 0 means no limit
	UploadSpeedLimit   int64 // <= 0 means no limit
	NoAdd              bool  // if true, brush and other tasks will NOT add any torrents to client
}

type TorrentOption struct {
	Name               string
	Category           string
	Tags               []string
	DownloadSpeedLimit int64
	UploadSpeedLimit   int64
	Paused             bool
}

type Client interface {
	GetTorrents(state string, category string, showAll bool) ([]Torrent, error)
	AddTorrent(torrentContent []byte, option *TorrentOption, meta map[string](int64)) error
	ModifyTorrent(infoHash string, option *TorrentOption, meta map[string](int64)) error
	DeleteTorrents(infoHashes []string, deleteFiles bool) error
	TorrentRootPathExists(rootFolder string) bool
	PurgeCache()
	GetStatus() (*Status, error)
	GetName() string
	GetClientConfig() *config.ClientConfigStruct
	SetConfig(variable string, value string) error
	GetConfig(variable string) (string, error)
}

type RegInfo struct {
	Name    string
	Creator func(string, *config.ClientConfigStruct, *config.ConfigStruct) (Client, error)
}

type ClientCreator func(*RegInfo) (Client, error)

var (
	Registry []*RegInfo = make([]*RegInfo, 0)
)

func Register(regInfo *RegInfo) {
	Registry = append(Registry, regInfo)
}

func Find(name string) (*RegInfo, error) {
	for _, item := range Registry {
		if item.Name == name {
			return item, nil
		}
	}
	return nil, fmt.Errorf("didn't find client %q", name)
}

func ClientExists(name string) bool {
	clientConfig := config.GetClientConfig(name)
	return clientConfig != nil
}

func CreateClient(name string) (Client, error) {
	clientConfig := config.GetClientConfig(name)
	if clientConfig == nil {
		return nil, fmt.Errorf("client %s not existed", name)
	}
	regInfo, err := Find(clientConfig.Type)
	if err != nil {
		return nil, fmt.Errorf("unsupported client type %s", clientConfig.Type)
	}
	return regInfo.Creator(name, clientConfig, config.Get())
}

func GenerateNameWithMeta(name string, meta map[string](int64)) string {
	str := name
	first := true
	for key, value := range meta {
		if value == 0 {
			continue
		}
		if first {
			str += "__meta."
			first = false
		} else {
			str += "."
		}
		str += fmt.Sprintf("%s_%d", key, value)
	}
	return str
}

func ParseMetaFromName(fullname string) (name string, meta map[string](int64)) {
	metaStrReg := regexp.MustCompile(`^(?P<name>.*?)__meta.(?P<meta>[._a-zA-Z0-9]+)$`)
	metaStrMatch := metaStrReg.FindStringSubmatch(fullname)
	if metaStrMatch != nil {
		name = metaStrMatch[metaStrReg.SubexpIndex("name")]
		meta = make(map[string]int64)
		ms := metaStrMatch[metaStrReg.SubexpIndex("meta")]
		metas := strings.Split(ms, ".")
		for _, s := range metas {
			kvs := strings.Split(s, "_")
			if len(kvs) >= 2 {
				v := utils.ParseInt(kvs[1])
				if v != 0 {
					meta[kvs[0]] = v
				}
			}
		}
	} else {
		name = fullname
	}
	return
}

func TorrentStateIconText(torrent *Torrent) string {
	switch torrent.State {
	case "downloading":
		process := int64(float64(torrent.SizeCompleted) / float64(torrent.Size) * 100)
		return fmt.Sprint("↓", process, "%")
	case "seeding":
		return "↑U"
	case "paused":
		return "-P" // may be unicode symbol ⏸
	case "completed":
		return "✓C"
	}
	return "-"
}
func init() {
}

func (torrent *Torrent) GetSiteFromTag() string {
	for _, tag := range torrent.Tags {
		if strings.HasPrefix(tag, "site:") {
			return tag[5:]
		}
	}
	return ""
}

func GenerateTorrentTagFromSite(site string) string {
	return "site:" + site
}

func PrintTorrents(torrents []Torrent, filter string) {
	fmt.Printf("%-40s  %40s  %10s  %6s  %12s  %12s  %25s\n", "Name", "InfoHash", "Size", "State", "↓S", "↑S", "Tracker")
	for _, torrent := range torrents {
		if filter != "" && !utils.ContainsI(torrent.Name, filter) && !utils.ContainsI(torrent.InfoHash, filter) {
			continue
		}
		name := torrent.Name
		utils.PrintStringInWidth(name, 40, true)
		fmt.Printf("  %40s  %10s  %6s  %10s/s  %10s/s  %25s\n",
			torrent.InfoHash,
			utils.BytesSize(float64(torrent.Size)),
			TorrentStateIconText(&torrent),
			utils.BytesSize(float64(torrent.DownloadSpeed)),
			utils.BytesSize(float64(torrent.UploadSpeed)),
			torrent.TrackerDomain,
		)
	}
}
