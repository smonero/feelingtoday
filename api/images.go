package handler

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var apiKey = os.Getenv("FLICKR_API_KEY")

var animals = []string{
	"monkey",
	"dog",
	"cat",
	"squirrel",
	"bird",
	"gorilla",
	"penguin",
	"cow",
	"chicken",
	"lemur",
}

var emotions = []string{
	"happy",
	"sad",
	"funny",
	"angry",
	"eating",
	"sleeping",
	"crazy",
	"laughing",
	"yawning",
}

type RawPhotosResponse struct {
	Photos struct {
		Photo []struct {
			URL string `json:"url_q"`
		} `json:"photo"`
	} `json:"photos"`
}

func searchPhotos(animal string, emotion string) (*Response, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.flickr.com/services/rest/", nil)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Add("method", "flickr.photos.search")
	q.Add("format", "json")
	q.Add("extras", "url_q")
	q.Add("safe_search", "1")
	q.Add("nojsoncallback", "1")
	q.Add("sort", "relevance")
	q.Add("per_page", "15")
	q.Add("api_key", apiKey)
	query := animal + " " + emotion
	q.Add("text", query)

	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parsed := &RawPhotosResponse{}

	if err := json.Unmarshal(b, parsed); err != nil {
		return nil, err
	}

	ret := &Response{}

	for _, photo := range parsed.Photos.Photo {
		ret.Photos = append(ret.Photos, photo.URL)
		ret.Emotions = append(ret.Emotions, emotion)
	}

	return ret, nil
}

type Response struct {
	Photos   []string `json:"photos"`
	Emotions []string `json:"emotions"`
}

func Handler(w http.ResponseWriter, r *http.Request) {

	// Seed the RNG.
	rng := rand.New(rand.NewSource(time.Now().Unix()))

	// Pick a random animal.
	animal := animals[rng.Intn(len(animals))]

	// Pick some random emotions and search for them.
	ch := make(chan *Response)
	var wg sync.WaitGroup
	for i, idx := range rng.Perm(len(emotions)) {
		wg.Add(1)
		go func(emotion string) {
			defer wg.Done()

			resp, err := searchPhotos(animal, emotion)
			if err != nil {
				return
			}
			ch <- resp

		}(emotions[idx])

		// why is this 4?
		if i == 4 {
			break
		}
	}

	// Wait for stuff to complete.
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Read all the results.
	allPhotos := []string{}
	for result := range ch {
		allPhotos = append(allPhotos, result.Photos...)
	}

	// Shuffle results and take the first 9 for the grid.
	totalResp := &Response{}
	for i, idx := range rng.Perm(len(allPhotos)) {
		totalResp.Photos = append(totalResp.Photos, allPhotos[idx])

		if i == 8 {
			break
		}
	}

	// Respond!
	ret, err := json.Marshal(totalResp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(ret)
}
