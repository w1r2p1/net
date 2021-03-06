package monitor

import (
	"encoding/json"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/skycoin/net/skycoin-messenger/factory"
	"github.com/skycoin/skycoin/src/cipher"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
	"os"
	"sort"
	"path/filepath"
	"github.com/skycoin/skycoin/src/util/file"
	"strconv"
)

type Conn struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	SendBytes   uint64 `json:"send_bytes"`
	RecvBytes   uint64 `json:"recv_bytes"`
	LastAckTime int64  `json:"last_ack_time"`
	StartTime   int64  `json:"start_time"`
}
type NodeServices struct {
	Type        string `json:"type"`
	Addr        string `json:"addr"`
	SendBytes   uint64 `json:"send_bytes"`
	RecvBytes   uint64 `json:"recv_bytes"`
	LastAckTime int64  `json:"last_ack_time"`
	StartTime   int64  `json:"start_time"`
}
type App struct {
	Index      int      `json:"index"`
	Key        string   `json:"key"`
	Attributes []string `json:"attributes"`
}

var (
	NULL = "null"
)
var (
	BAD_REQUEST  = 400
	NOT_FOUND    = 404
	SERVER_ERROR = 500
)

type Monitor struct {
	factory *factory.MessengerFactory
	address string
	srv     *http.Server

	code    string
	version string

	configs      map[string]*Config
	configsMutex sync.RWMutex
}

func New(f *factory.MessengerFactory, addr, code, version string) *Monitor {
	return &Monitor{
		factory: f,
		address: addr,
		srv:     &http.Server{Addr: addr},
		code:    code,
		version: version,
		configs: make(map[string]*Config),
	}
}

func (m *Monitor) Close() error {
	return m.srv.Close()
}
func (m *Monitor) Start(webDir string) {
	http.Handle("/", http.FileServer(http.Dir(webDir)))
	http.HandleFunc("/conn/getAll", bundle(m.getAllNode))
	http.HandleFunc("/conn/getNode", bundle(m.getNode))
	http.HandleFunc("/conn/setNodeConfig", bundle(m.setNodeConfig))
	http.HandleFunc("/conn/getNodeConfig", bundle(m.getNodeConfig))
	http.HandleFunc("/conn/saveClientConnection", bundle(m.SaveClientConnection))
	http.HandleFunc("/conn/removeClientConnection", bundle(m.RemoveClientConnection))
	http.HandleFunc("/conn/editClientConnection", bundle(m.EditClientConnection))
	http.HandleFunc("/conn/getClientConnection", bundle(m.GetClientConnection))
	http.HandleFunc("/node", bundle(requestNode))
	go func() {
		if err := m.srv.ListenAndServe(); err != nil {
			log.Printf("http server: ListenAndServe() error: %s", err)
		}
	}()
	log.Debugf("http server listen on %s", m.address)
}

func bundle(fn func(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err, code := fn(w, r)
		if err != nil {
			if code == 0 {
				code = SERVER_ERROR
			}
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	}
}

func requestNode(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if r.Method != "POST" {
		code = BAD_REQUEST
		err = errors.New("please use post method")
		return
	}
	addr := r.FormValue("addr")
	res, err := http.PostForm(addr, r.PostForm)
	if err != nil {
		if res != nil {
			return result, err, res.StatusCode
		}
		return result, err, 404
	}
	defer res.Body.Close()
	result, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return result, err, SERVER_ERROR
	}
	return
}

func (m *Monitor) getAllNode(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	cs := make([]Conn, 0)
	m.factory.ForEachAcceptedConnection(func(key cipher.PubKey, conn *factory.Connection) {
		now := time.Now().Unix()
		content := Conn{
			Key:         key.Hex(),
			SendBytes:   conn.GetSentBytes(),
			RecvBytes:   conn.GetReceivedBytes(),
			StartTime:   now - conn.GetConnectTime(),
			LastAckTime: now - conn.GetLastTime()}
		if conn.IsTCP() {
			content.Type = "TCP"
		} else {
			content.Type = "UDP"
		}
		cs = append(cs, content)
	})
	result, err = json.Marshal(cs)
	if err != nil {
		code = SERVER_ERROR
		return
	}
	return
}

func (m *Monitor) getNode(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if r.Method != "POST" {
		code = BAD_REQUEST
		err = errors.New("please use post method")
		return
	}
	key, err := cipher.PubKeyFromHex(r.FormValue("key"))
	if err != nil {
		code = BAD_REQUEST
		return
	}
	c, ok := m.factory.GetConnection(key)
	if !ok {
		code = NOT_FOUND
		err = errors.New("No connection is found")
		return
	}
	now := time.Now().Unix()
	nodeService := NodeServices{
		SendBytes:   c.GetSentBytes(),
		RecvBytes:   c.GetReceivedBytes(),
		StartTime:   now - c.GetConnectTime(),
		LastAckTime: now - c.GetLastTime()}
	if c.IsTCP() {
		nodeService.Type = "TCP"
	} else {
		nodeService.Type = "UDP"
	}
	v, ok := c.LoadContext("node-api")
	if ok {
		webPort, ok := v.(string)
		if ok && len(webPort) > 1 {
			var host, port string
			host, _, err = net.SplitHostPort(c.GetRemoteAddr().String())
			if err != nil {
				code = SERVER_ERROR
				return
			}
			_, port, err = net.SplitHostPort(webPort)
			if err != nil {
				code = SERVER_ERROR
				return
			}
			nodeService.Addr = net.JoinHostPort(host, port)
		}
	}
	result, err = json.Marshal(nodeService)
	if err != nil {
		code = SERVER_ERROR
		return
	}
	return
}

type Config struct {
	DiscoveryAddresses []string
}

func (m *Monitor) setNodeConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if r.Method != "POST" {
		code = BAD_REQUEST
		err = errors.New("please use post method")
		return
	}
	key := r.FormValue("key")
	data := []byte(r.FormValue("data"))
	var config *Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return
	}
	m.configsMutex.Lock()
	m.configs[key] = config
	m.configsMutex.Unlock()
	result = []byte("true")
	return
}

func (m *Monitor) getNodeConfig(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	if r.Method != "POST" {
		code = BAD_REQUEST
		err = errors.New("please use post method")
		return
	}
	key := r.FormValue("key")
	m.configsMutex.Lock()
	defer m.configsMutex.Unlock()
	result, err = json.Marshal(m.configs[key])
	return
}

type ClientConnection struct {
	Label   string `json:"label"`
	NodeKey string `json:"nodeKey"`
	AppKey  string `json:"appKey"`
	Count   int    `json:"count"`
}
type clientConnectionSlice []ClientConnection

func (c clientConnectionSlice) Len() int           { return len(c) }
func (c clientConnectionSlice) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c clientConnectionSlice) Less(i, j int) bool { return c[i].Count > c[j].Count }
func (c clientConnectionSlice) Exist(rf ClientConnection) bool {
	for k, v := range c {
		if v.AppKey == rf.AppKey && v.NodeKey == rf.NodeKey {
			c[k].Count++
			return true
		}
	}
	return false
}

var sshClient = filepath.Join(file.UserHome(), ".skywire", "manager", "sshClient.json")
var socketClient = filepath.Join(file.UserHome(), ".skywire", "manager", "socketClient.json")
var clientLimit = 5

func (m *Monitor) SaveClientConnection(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	data := r.FormValue("data")
	path := r.FormValue("client")
	config := ClientConnection{}
	err = json.Unmarshal([]byte(data), &config)
	if err != nil {
		return
	}
	switch path {
	case "ssh":
		path = sshClient
		break
	case "socket":
		path = socketClient
	}
	cfs, err := readConfig(path)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	size := len(cfs)
	isExist := false
	if size == clientLimit {
		isExist = cfs.Exist(config)
		if !isExist {
			cfs[4] = config
		}
	} else if size > 0 && size < clientLimit {
		isExist = cfs.Exist(config)
		if !isExist {
			cfs = append(cfs, config)
		}
	} else {
		cfs = append(cfs, config)
	}
	sort.Sort(cfs)
	err = saveClientFile(cfs, path)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (m *Monitor) GetClientConnection(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	client := r.FormValue("client")
	switch client {
	case "ssh":
		client = sshClient
		break
	case "socket":
		client = socketClient
	}
	cf, err := readConfig(client)
	result, err = json.Marshal(cf)
	return
}

func (m *Monitor) RemoveClientConnection(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	path := r.FormValue("client")
	index, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		return
	}
	path = getFilePath(path)
	cfs, err := readConfig(path)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	cfs = append(cfs[:index], cfs[index+1:]...)
	err = saveClientFile(cfs, path)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func (m *Monitor) EditClientConnection(w http.ResponseWriter, r *http.Request) (result []byte, err error, code int) {
	path := r.FormValue("client")
	label := r.FormValue("label")
	index, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		return
	}
	path = getFilePath(path)
	cfs, err := readConfig(path)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	cfs[index].Label = label
	err = saveClientFile(cfs, path)
	if err != nil {
		return
	}
	result = []byte("true")
	return
}

func readConfig(path string) (cfs clientConnectionSlice, err error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = json.Unmarshal(fb, &cfs)
	if err != nil {
		return
	}
	return
}

func saveClientFile(data interface{}, path string) (err error) {
	d, err := json.Marshal(data)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, d, 0600)
	return
}

func getFilePath(client string) string {
	switch client {
	case "ssh":
		client = sshClient
		break
	case "socket":
		client = socketClient
	}
	return client
}
