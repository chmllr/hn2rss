package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/feeds"
)

const api = "https://hacker-news.firebaseio.com/v0"

func handler(w http.ResponseWriter, r *http.Request) {
	score := 250
	vals := r.URL.Query()
	if points := vals["points"]; len(points) > 0 {
		if s, err := strconv.ParseInt(points[0], 10, 16); err == nil {
			score = int(s)
		}
	}
	start := time.Now()
	items, err := fetch(score)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	rss, err := item2RSS(score, items)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", rss)
	log.Println("request took", time.Since(start))
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}

func item2RSS(score int, items []item) (string, error) {
	feed := &feeds.Feed{
		Title:       fmt.Sprintf("Hacker News %d", score),
		Link:        &feeds.Link{Href: "https://github.com/chmllr/hn2rss"},
		Description: "Top Hacker News Stories",
		Author:      &feeds.Author{Name: "Christian Müller", Email: "@drmllr"},
		Created:     time.Now(),
	}

	feed.Items = make([]*feeds.Item, len(items))
	for i, item := range items {
		feed.Items[i] = &feeds.Item{
			Title:       item.Title,
			Link:        &feeds.Link{Href: item.Url},
			Description: fmt.Sprintf("%d comments: https://news.ycombinator.com/item?id=%d", item.Comments, item.ID),
			Author:      &feeds.Author{Name: item.Author},
			Created:     time.Unix(item.Time, 0),
		}
	}

	return feed.ToRss()
}

func fetch(score int) ([]item, error) {
	fd, err := feed(score)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch feed: %v", err)
	}
	arr := make([]*item, len(fd))
	var wg sync.WaitGroup
	wg.Add(len(fd))
	for i, id := range fd {
		go func(i, id int) {
			item, err := story(id)
			if err != nil {
				log.Printf("couldn't fetch item %d: %v\n", id, err)
			}
			if item.Score >= score {
				arr[i] = &item
			}
			wg.Done()
		}(i, id)
	}
	wg.Wait()
	res := []item{}
	for _, v := range arr {
		if v != nil {
			res = append(res, *v)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Time > res[j].Time })
	return res, nil
}

type ids []int

func feed(score int) (ids, error) {
	var res ids
	resp, err := http.Get(api + "/topstories.json")
	if err != nil {
		return res, fmt.Errorf("couldn't fetch topstories: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, fmt.Errorf("couldn't read response: %v", err)
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return res, fmt.Errorf("couldn't unmarshal response: %v", err)
	}
	return res, nil
}

type item struct {
	ID       int    `json:"id"`
	Time     int64  `json:"time"`
	Score    int    `json:"score"`
	Comments int    `json:"descendants"`
	Title    string `json:"title"`
	Author   string `json:"by"`
	Url      string `json:"url"`
}

func story(id int) (item, error) {
	var res item
	resp, err := http.Get(fmt.Sprintf("%s/item/%d.json", api, id))
	if err != nil {
		return res, fmt.Errorf("couldn't fetch story: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, fmt.Errorf("couldn't read response: %v", err)
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return res, fmt.Errorf("couldn't unmarshal response: %v", err)
	}
	return res, nil
}
