package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

type GenericExtractor struct{}

func (g *GenericExtractor) CanHandle(url string) bool {
	return true
}

func (g *GenericExtractor) Extract(ctx context.Context, seshuJob types.SeshuJob, scraper ScrapingService) ([]types.EventInfo, string, error) {

	var localPrompt string
	mode := ctx.Value("MODE")
	action := ctx.Value("ACTION")

	html, err := scraper.GetHTMLFromURL(seshuJob, 4500, true, "")
	if err != nil {
		return nil, "", err
	}

	if mode == constants.SESHU_MODE_ONBOARD {

		var response string

		if action == "init" {
			localPrompt = GetSystemPrompt(false)
		} else if action == "rs" {
			localPrompt = GetSystemPrompt(true)
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return nil, "", err
		}
		bodyHtml, err := doc.Find("body").Html()
		if err != nil {
			return nil, "", err
		}

		markdown, err := converter.ConvertString(bodyHtml)
		if err != nil {
			return nil, "", err
		}

		lines := strings.Split(markdown, "\n")
		var filtered []string
		for i, line := range lines {
			if line != "" && i < 1500 {
				filtered = append(filtered, line)
			}
		}

		jsonPayload, err := json.Marshal(filtered)
		if err != nil {
			return nil, "", err
		}

		_, response, err = CreateChatSession(string(jsonPayload), localPrompt)
		if err != nil {
			return nil, "", err
		}

		var events []types.EventInfo
		err = json.Unmarshal([]byte(response), &events)
		if err != nil {
			return nil, "", err
		}

		return events, html, err
	}

	// TO DO: For non-onboard modes, implement logic

	return []types.EventInfo{}, html, nil
}
