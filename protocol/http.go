package protocol

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// HTTP implements Protocol for HTTP
type HTTP struct {
	absProtocol
}

func NewHTTP(r ServeResource) *HTTP {
	return &HTTP{absProtocol{r: r}}
}

func (p *HTTP) Serve(addr string) error {
	return http.ListenAndServe(addr, p)
}

func (p *HTTP) GetFromRemote(addr, groupName, key string) (val []byte, err error) {
	resp, err := http.Get("http://" + addr + "/" + groupName + "/" + key)
	if err != nil {
		return nil, err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", resp.Status)
	}
	val, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response body: %v", err)
	}

	return val, nil
}

func (p *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(r.URL.Path, "/", 3)[1:] // URL format: /<groupName>/<key>
	if len(parts) < 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]

	val, err := p.r.Get(groupName, key)
	if val == nil {
		http.Error(w, "no such rescue: "+groupName+"-"+key, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(val)
}
