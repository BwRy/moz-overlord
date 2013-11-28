package main

import (
	"github.com/st3fan/moz-go-bugzilla"
	"math"
	"os"
	"strconv"
	"time"
)

// Bugzilla

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func calculateBugRiskScore(b bugzilla.Bug) float64 {
	rating := 0.0

	// Map the bug's security rating to our risk rating
	keywordMappings := map[string]float64{
		"sec-critical": RISK_RATING_CRITICAL,
		"sec-high":     RISK_RATING_HIGH,
		"sec-moderate": RISK_RATING_MODERATE,
		"sec-low":      RISK_RATING_LOW,
	}

	for keyword, r := range keywordMappings {
		if stringInSlice(keyword, b.Keywords) {
			rating = r
			break
		}
	}

	// Default to MODERATE if we are unable to find a rating in a the bug
	if rating == 0 {
		rating = RISK_RATING_MODERATE
	}

	// Boost the rating a bit based on the age of the bug
	age := b.Age().Hours() / 24
	if age > 7 {
		rating *= 1.05
	} else if age > 28 {
		rating *= 1.1
	} else if age > 56 {
		rating *= 1.25
	} else if age > 365 {
		rating *= 1.5
	}

	return math.Min(rating, RISK_RATING_CRITICAL)
}

func severityFromBug(b bugzilla.Bug) string {
	keywordMappings := map[string]string{
		"sec-critical": "Critical",
		"sec-high":     "High",
		"sec-moderate": "Moderate",
		"sec-low":      "Low",
	}

	for keyword, r := range keywordMappings {
		if stringInSlice(keyword, b.Keywords) {
			return r
		}
	}

	return "Unknown"
}

func CollectBugzillaResults(p *Site) ([]DataSourceResults, error) {
	bz := bugzilla.NewBugzilla()

	if len(os.Getenv("BZ_USERNAME")) != 0 && len(os.Getenv("BZ_PASSWORD")) != 0 {
		_, err := bz.Login(os.Getenv("BZ_USERNAME"), os.Getenv("BZ_PASSWORD"))
		if err != nil {
			return nil, err
		}
	}

	result := []DataSourceResults{}

	// Find web security bugs that have a proper product/component

	webSecurityBugs := make(map[int]bugzilla.Bug)

	bugs, err := bz.GetBugs().
		Product("Websites").Component(p.Host).
		Status("NEW", "UNCONFIRMED").
		Advanced("bug_group", "equals", "websites-security").
		Execute()
	if err != nil {
		return nil, err
	}
	for _, bug := range bugs {
		webSecurityBugs[bug.Id] = bug
	}

	// Find web security bugs that have a [site:foo.com] whiteboard tag

	bugs, err = bz.GetBugs().
		Status("NEW", "UNCONFIRMED").
		//Product("mozilla.org").Component("Security Assurance").
		Advanced("bug_group", "equals", "websites-security").
		Advanced("status_whiteboard", "substring", "[site:"+p.Host+"]").
		Execute()
	if err != nil {
		return nil, err
	}
	for _, bug := range bugs {
		webSecurityBugs[bug.Id] = bug
	}

	// Rate all the bugs

	if len(webSecurityBugs) > 0 {
		results := DataSourceResults{Date: time.Now(), Source: "bugzilla-outstanding-web-security-bugs"}
		for _, bug := range webSecurityBugs {
			issue := DataSourceIssue{
				Description: strconv.Itoa(bug.Id) + " " + bug.Summary,
				Score:       calculateBugRiskScore(bug),
				DetailLink:  "https://bugzilla.mozilla.org/show_bug.cgi?id=" + strconv.Itoa(bug.Id),
				Severity:    severityFromBug(bug),
			}
			results.Issues = append(results.Issues, issue)
			results.Score += issue.Score
		}

		result = append(result, results)
	}

	// Outstanding Bug Bounty Bugs

	return result, nil
}
