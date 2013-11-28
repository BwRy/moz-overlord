package main

import (
	"fmt"
	"github.com/st3fan/moz-go-minion-client"
	"log"
	"os"
)

func calculateMinionIssueRiskScore(issue minion.ScanSessionIssue) float64 {
	severityMappings := map[string]float64{
		"High":   RISK_RATING_HIGH,
		"Medium": RISK_RATING_MODERATE,
		"Low":    RISK_RATING_LOW,
	}

	for severity, rating := range severityMappings {
		if issue.Severity == severity {
			return rating
		}
	}

	return 0.0
}

func CollectMinionResults(p *Site) ([]DataSourceResults, error) {
	// Grab the sites and then find out relevant ones

	client, err := minion.NewClient("http://localhost:8080", os.Getenv("MINION_API_USER"), os.Getenv("MINION_API_KEY"))
	if err != nil {
		return nil, err
	}

	sites, err := client.GetSites()
	if err != nil {
		return nil, err
	}

	result := []DataSourceResults{}

	for _, site := range sites {
		if site.URL == "http://"+p.Host || site.URL == "https://"+p.Host {
			for _, plan := range site.Plans {
				scans, err := client.GetScans(site.Id, plan, 1)
				log.Printf("Scan %+v", scans)
				if err != nil {
					return nil, err
				}
				if len(scans) > 0 {
					for _, scan := range scans {
						if scan.State == "FINISHED" {
							scan, err := client.GetScan(scan.Id)
							if err != nil {
								return nil, err
							}

							results := DataSourceResults{Source: "minion"}

							for _, session := range scan.Sessions {
								if len(session.Issues) > 0 {
									for _, issue := range session.Issues {
										if issue.Severity == "High" || issue.Severity == "Medium" || issue.Severity == "Low" {
											overlordIssue := DataSourceIssue{
												Description: issue.Summary,
												Score:       calculateMinionIssueRiskScore(issue),
												Severity:    issue.Severity,
												DetailLink:  fmt.Sprintf("https://minion-dev.mozillalabs.com/#!/scan/%s/issue/%s", scan.Id, issue.Id),
											}
											results.Score += overlordIssue.Score
											results.Issues = append(results.Issues, overlordIssue)
										}
									}
								}
							}

							result = append(result, results)

							break
						}
					}
				}
			}
		}
	}

	return result, nil
}
