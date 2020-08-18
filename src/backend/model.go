package main

type TODO struct {
	ID        int    `json:"-"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Order     int    `json:"order"`
	URL       string `json:"url"`
}
