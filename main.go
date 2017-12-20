package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"sync"
)

const api = "https://hacker-news.firebaseio.com/v0/"

func handler(w http.ResponseWriter, r *http.Request) {
	score := int64(250)
	vals := r.URL.Query()
	if points := vals["points"]; len(points) > 0 {
		if s, err := strconv.ParseInt(points[0], 10, 16); err == nil {
			score = s
		}
	}
	fmt.Fprintf(w, "%+v", score)
}

func main() {
	// http.HandleFunc("/", handler)
	// http.ListenAndServe(":8080", nil)

	items, _ := fetch(250)
	s, _ := json.Marshal(items)
	fmt.Println(string(s))
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
				fmt.Printf("couldn't fetch item %d: %v\n", id, err)
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
	sort.Slice(res, func(i, j int) bool { return res[i].Time < res[j].Time })
	return res, nil
}

type ids []int

func feed(score int) (ids, error) {
	var res ids
	resp, err := http.Get(api + "topstories.json")
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
	Time     int    `json:"time"`
	Score    int    `json:"score"`
	Comments int    `json:"descendants"`
	Title    string `json:"title"`
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
