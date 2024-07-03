package schedule

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Timeblock struct {
	StartAt *time.Time
	EndAt   *time.Time
}

type Blackouts []Timeblock

const (
	blackoutsPageURL = "https://oblenergo.cv.ua/shutdowns"

	minGroupNumber = 1
	maxGroupNumber = 18

	tableValuePowerIsOn = "з"
)

func loadBlackoutsPage() (string, error) {
	resp, err := http.Get(blackoutsPageURL)
	if err != nil {
		return "", fmt.Errorf("send GET request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response status code: %v", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	return string(respBody), nil
}

func findChildByCriteriaFunc(node *html.Node, criteriaFunc func(child *html.Node) bool) *html.Node {
	child := node.FirstChild
	if child == nil {
		return nil
	}

	for {
		if criteriaFunc(child) {
			return child
		}

		if child.NextSibling == nil {
			break
		}
		child = child.NextSibling
	}
	return nil
}

func getElementID(node *html.Node) string {
	for _, attr := range node.Attr {
		if attr.Key == "id" {
			return attr.Val
		}
	}
	return ""
}

func findChildElementWithID(node *html.Node, id string) *html.Node {
	return findChildByCriteriaFunc(node, func(child *html.Node) bool {
		return getElementID(child) == id
	})
}

func findChildElementOfType(node *html.Node, elType string) *html.Node {
	return findChildByCriteriaFunc(node, func(child *html.Node) bool {
		return child.Data == elType
	})
}

func dayStartTimeCV() time.Time {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		log.Fatalln("failed to load timezone", err)
	}
	return time.Now().In(loc)
}

func LoadBlackouts(group int) (Blackouts, error) {
	if group < minGroupNumber || group > maxGroupNumber {
		return nil, fmt.Errorf("invalid group number, must be within [%v, %v] range", minGroupNumber, maxGroupNumber)
	}

	page, err := loadBlackoutsPage()
	if err != nil {
		return nil, fmt.Errorf("load blackouts page: %w", err)
	}

	rootNode, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return nil, fmt.Errorf("parse page HTML: %w", err)
	}

	htmlNode := findChildElementOfType(rootNode, "html").NextSibling
	if htmlNode == nil {
		return nil, fmt.Errorf("missing 'html' tag on the page")
	}

	bodyNode := findChildElementOfType(htmlNode, "body")
	if bodyNode == nil {
		return nil, fmt.Errorf("missing 'body' tag on the page")
	}

	mainNode := findChildElementOfType(bodyNode, "main")
	if mainNode == nil {
		return nil, fmt.Errorf("can not find 'main' tag on the page")
	}

	gsvNode := findChildElementWithID(mainNode, "gsv")
	if gsvNode == nil {
		return nil, fmt.Errorf("can not find the schedule table on the page")
	}

	innerDiv := findChildElementOfType(gsvNode, "div")
	if innerDiv == nil {
		return nil, fmt.Errorf("unexpected schedule table structure")
	}

	groupRowElID := fmt.Sprint("inf", group)
	groupRow := findChildElementWithID(innerDiv, groupRowElID)
	if groupRow == nil {
		return nil, fmt.Errorf("can not find a row for the requested group number")
	}

	var blackouts Blackouts

	startTime := dayStartTimeCV()
	endTime := startTime.Add(1 * time.Hour)

	currTimeblock := Timeblock{}

	groupColumn := groupRow.FirstChild
	for {
		if strings.TrimSpace(groupColumn.FirstChild.Data) != tableValuePowerIsOn {

			if currTimeblock.StartAt == nil {
				currTimeblock.StartAt = &startTime
			}

		} else if currTimeblock.StartAt != nil {
			currTimeblock.EndAt = &startTime
			blackouts = append(blackouts, currTimeblock)

			currTimeblock = Timeblock{}
		}

		if groupColumn.NextSibling == nil {
			break
		}
		groupColumn = groupColumn.NextSibling

		startTime = endTime
		endTime = endTime.Add(1 * time.Hour)
	}

	return blackouts, nil
}
