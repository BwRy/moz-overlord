package main

import (
	"time"
)

type Group struct {
	Name  string
	Sites []string
}

type User struct {
	Email        string
	LastLogin    time.Time
	DefaultGroup string
}

type ResultsReference struct {
	ResultsId interface{} `bson:"resultsId"`
	Date      time.Time   `bson:"date"`
	Score     float64     `bson:"score"`
}

type Site struct {
	Host          string             `bson:"host"`
	RecentResults []ResultsReference `bson:"recentResults"`
}

type DataSourceIssue struct {
	Description string  `bson:"description"`
	Severity    string  `bson:"severity"`
	DetailLink  string  `bson:"detailLink"`
	Score       float64 `bson:"score"`
}

type DataSourceResults struct {
	Source string            `bson:"source"`
	Date   time.Time         `bson:"date"`
	Issues []DataSourceIssue `bson:"issues"`
	Score  float64           `bson:"score"`
}

type Results struct {
	Id                interface{}         `bson:"_id"`
	Site              string              `bson:"site"`
	Date              time.Time           `bson:"date"`
	Score             float64             `bson:"score"`
	DataSourceResults []DataSourceResults `bson:"dataSourceResults"`
}
