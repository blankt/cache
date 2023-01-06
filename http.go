package cache

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"cache/cachepb"
	"cache/consistenthash"
	"google.golang.org/protobuf/proto"
)

const (
	defaultBaseUrl  = "/_distrcache/"
	defaultReplicas = 50
)

var (
	_ PeerPicker = (*HttpPool)(nil)
	_ PeerGetter = (*httpGetter)(nil)
)

type HttpPool struct {
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter
}

func NewHttpPool(self string) *HttpPool {
	return &HttpPool{
		self:        self,
		basePath:    defaultBaseUrl,
		httpGetters: make(map[string]*httpGetter),
	}
}

func (h *HttpPool) Log(format string, values ...interface{}) {
	log.Printf("[server: %v] %v", h.self, fmt.Sprintf(format, values...))
}

//url : http://localhost:8080/_distrcache/groupName/key
func (h *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, defaultBaseUrl) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	h.Log("%s %s", r.Method, r.URL.Path)

	parts := strings.SplitN(r.URL.Path[len(h.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group"+groupName, http.StatusNotFound)
	}

	data, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	body, err := proto.Marshal(&cachepb.Response{Value: data.ByteSlice()})
	if err != nil {
		http.Error(w, fmt.Errorf("encode response err:%v", err).Error(), http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

func (h *HttpPool) Set(peers ...string) {
	h.mu.Lock()
	h.mu.Unlock()
	h.peers = consistenthash.New(defaultReplicas, nil)
	h.peers.Add(peers...)
	for _, peer := range peers {
		httpGetter := &httpGetter{
			baseUrl: peer + defaultBaseUrl,
		}
		h.httpGetters[peer] = httpGetter
	}
}

func (h *HttpPool) PickPeer(key string) (PeerGetter, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if peer := h.peers.Get(key); peer != "" && h.self != peer {
		log.Printf("get peer %v", peer)
		return h.httpGetters[peer], true
	}
	return nil, false
}

type httpGetter struct {
	baseUrl string
}

func (h *httpGetter) Get(in *cachepb.Request, out *cachepb.Response) error {
	requestUrl := fmt.Sprintf("%v%v/%v", h.baseUrl, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))

	response, err := http.Get(requestUrl)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad request")
		return err
	}

	data, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return err
	}

	if err = proto.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode response body err:%v", err)
	}

	return nil
}
