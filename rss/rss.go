// Simple gmail ATOM parser
package rss

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/svagner/gmail2go/logger"
)

type Author struct {
	Name  string `xml:"name"`
	Email string `xml:"email"`
}

type Entry struct {
	Title    string `xml:"title"`
	Summary  string `xml:"summary"`
	Modified string `xml:"modified"`
	Id       string `xml:"id"`
	Author   Author `xml:"author"`
}

// Parses modification time string to time structure
func (e *Entry) ModifiedTime() (time.Time, error) {
	return time.Parse(time.RFC3339, e.Modified)
}

// Returns a list of Entry objects by parsing the url atom feed
func Read(url, user, pass string) ([]*Entry, error) {
	var myTransport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		ResponseHeaderTimeout: time.Second * 2,
	}

	var client = &http.Client{Transport: myTransport}

	req, err := http.NewRequest("GET", url, nil)
	logger.DebugPrint("Try to auth at ", url)
	req.SetBasicAuth(user, pass)
	resp, err := client.Do(req)

	if err != nil {
		logger.WarningPrint(err)
		return nil, err
	}
	logger.DebugPrint("RSS: ", resp.StatusCode)
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Received bad status code: %v", resp.StatusCode))
	}
	defer resp.Body.Close()

	text, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	logger.DebugPrint("Get response: ", string(text))

	return unmarshal(text)
}

func unmarshal(text []byte) (es []*Entry, err error) {
	var feed struct {
		Entries []*Entry `xml:"entry"`
	}
	err = xml.Unmarshal(text, &feed)
	if err != nil {
		return nil, err
	}

	return feed.Entries, nil
}
