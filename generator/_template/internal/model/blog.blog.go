package model

import "time"

type BlogPost struct {
	Title       string
	Slug        string
	Date        time.Time
	Author      string
	Description string
	Tags        []string
	Content     string
	HTMLContent string
	ReadTime    int
	HeroImage   string
}
